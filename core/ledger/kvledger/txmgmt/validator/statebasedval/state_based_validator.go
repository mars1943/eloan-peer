/*
Copyright IBM Corp. 2016 All Rights Reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		 http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package statebasedval

import (
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/rwsetutil"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/statedb"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/validator/statebasedval/cache"
	"github.com/hyperledger/fabric/core/ledger/kvledger/txmgmt/version"
	"github.com/hyperledger/fabric/core/ledger/util"
	"github.com/hyperledger/fabric/protos/common"
	"github.com/hyperledger/fabric/protos/ledger/rwset/kvrwset"
	"github.com/hyperledger/fabric/protos/peer"
	putils "github.com/hyperledger/fabric/protos/utils"
)

var logger = flogging.MustGetLogger("statevalidator")

//rongzer,王剑增加
var rListCache *cache.Cache
var curBlockStatistics *common.BlockStatistics

// Validator validates a tx against the latest committed state
// and preceding valid transactions with in the same block
type Validator struct {
	db statedb.VersionedDB
}

// NewValidator constructs StateValidator
func NewValidator(db statedb.VersionedDB) *Validator {
	return &Validator{db}
}

//validate endorser transaction
func (v *Validator) validateEndorserTX(envBytes []byte, doMVCCValidation bool, updates *statedb.UpdateBatch) (*rwsetutil.TxRwSet, peer.TxValidationCode, error) {
	// extract actions from the envelope message
	respPayload, err := putils.GetActionFromEnvelope(envBytes)
	if err != nil {
		return nil, peer.TxValidationCode_NIL_TXACTION, nil
	}

	//preparation for extracting RWSet from transaction
	txRWSet := &rwsetutil.TxRwSet{}

	// Get the Result from the Action
	// and then Unmarshal it into a TxReadWriteSet using custom unmarshalling

	if err = txRWSet.FromProtoBytes(respPayload.Results); err != nil {
		return nil, peer.TxValidationCode_INVALID_OTHER_REASON, nil
	}

	txResult := peer.TxValidationCode_VALID

	//mvccvalidation, may invalidate transaction
	if doMVCCValidation {
		if txResult, err = v.validateTx(txRWSet, updates); err != nil {
			return nil, txResult, err
		} else if txResult != peer.TxValidationCode_VALID {
			txRWSet = nil
		}
	}

	return txRWSet, txResult, err
}

// ValidateAndPrepareBatch implements method in Validator interface
func (v *Validator) ValidateAndPrepareBatch(block *common.Block, doMVCCValidation bool) (*statedb.UpdateBatch, error) {
	logger.Debugf("New block arrived for validation:%#v, doMVCCValidation=%t", block, doMVCCValidation)
	updates := statedb.NewUpdateBatch()
	logger.Debugf("Validating a block with [%d] transactions", len(block.Data.Data))

	// Committer validator has already set validation flags based on well formed tran checks
	txsFilter := util.TxValidationFlags(block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER])

	// Precaution in case committer validator has not added validation flags yet
	if len(txsFilter) == 0 {
		txsFilter = util.NewTxValidationFlags(len(block.Data.Data))
		block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
	}
	//rongzer,王剑，增加交易统计功能
	mapTransStatic := make(map[string]*common.TransactionStatistics)
	if curBlockStatistics == nil {

		curBlockStatistics = &common.BlockStatistics{}
		curBlockStatistics.List = make([]*common.TransactionStatistics, 0)

		//从DB中查询
		existValue, _ := v.db.GetState("qscc", "curBlockStatistics")
		if existValue != nil && len(existValue.Value) > 1 {
			proto.Unmarshal(existValue.Value, curBlockStatistics)
		}
	}

	curBlockStatistics.Number = block.GetHeader().Number
	vh, _ := v.db.GetLatestSavePoint()
	updates.GetOrCreateNsUpdates("qscc")
	mapStaticUpdates := updates.GetUpdates("qscc")

	//List转Map
	for _, transStatic := range curBlockStatistics.List {
		transStaticKey := transStatic.ChaincodeId + "-" + transStatic.Func + "-" + transStatic.ValidationCode
		mapTransStatic[transStaticKey] = transStatic
	}

	for txIndex, envBytes := range block.Data.Data {
		if txsFilter.IsInvalid(txIndex) {
			// Skiping invalid transaction
			logger.Warningf("Block [%d] Transaction index [%d] marked as invalid by committer. Reason code [%d]",
				block.Header.Number, txIndex, txsFilter.Flag(txIndex))
			continue
		}

		env, err := putils.GetEnvelopeFromBlock(envBytes)
		if err != nil {
			return nil, err
		}

		payload, err := putils.GetPayload(env)
		if err != nil {
			return nil, err
		}

		chdr, err := putils.UnmarshalChannelHeader(payload.Header.ChannelHeader)
		if err != nil {
			return nil, err
		}

		txType := common.HeaderType(chdr.Type)

		if txType != common.HeaderType_ENDORSER_TRANSACTION {
			logger.Debugf("Skipping mvcc validation for Block [%d] Transaction index [%d] because, the transaction type is [%s]",
				block.Header.Number, txIndex, txType)
			continue
		}

		txRWSet, txResult, err := v.validateEndorserTX(envBytes, doMVCCValidation, updates)

		if err != nil {
			return nil, err
		}

		txsFilter.SetFlag(txIndex, txResult)

		//txRWSet != nil => t is valid
		if txRWSet != nil {
			committingTxHeight := version.NewHeight(block.Header.Number, uint64(txIndex))
			addWriteSetToBatch(txRWSet, committingTxHeight, updates)
			txsFilter.SetFlag(txIndex, peer.TxValidationCode_VALID)
		}

		//rongzer,王剑增加，交易统计
		if txIndex == 0 {
			curBlockStatistics.BlockTime = chdr.Timestamp
		}

		//解释交易
		ccs := &peer.ChaincodeSpec{}
		proto.Unmarshal(chdr.Extension, ccs)
		strFunc := "func"

		tx, err := putils.GetTransaction(payload.Data)
		if tx != nil {
			chaincodeActionPayload, _ := putils.GetChaincodeActionPayload(tx.Actions[0].Payload)
			if chaincodeActionPayload != nil {
				chaincodeProposalPayload, _ := putils.GetChaincodeProposalPayload(chaincodeActionPayload.ChaincodeProposalPayload)
				if chaincodeProposalPayload != nil {
					chaincodeInvocationSpec := &peer.ChaincodeInvocationSpec{}
					err = proto.Unmarshal(chaincodeProposalPayload.Input, chaincodeInvocationSpec)

					if err == nil {
						if chaincodeInvocationSpec.ChaincodeSpec.Input != nil {
							for j, arg := range chaincodeInvocationSpec.ChaincodeSpec.Input.Args {
								strFunc += "," + string(arg)
								if j >= 1 {
									break
								}
							}
						}
					}
				}
			}
		}
		if err != nil {
			logger.Errorf("parse transaction aggs err %s", err)
		}

		validateCode := strconv.Itoa(int(txsFilter.Flag(txIndex)))
		v.calStatistics(mapStaticUpdates, mapTransStatic, vh, chdr.TxId, ccs.ChaincodeId.Name, strFunc, validateCode)

		if txsFilter.IsValid(txIndex) {
			logger.Debugf("Block [%d] Transaction index [%d] TxId [%s] marked as valid by state validator",
				block.Header.Number, txIndex, chdr.TxId)
		} else {
			logger.Warningf("Block [%d] Transaction index [%d] TxId [%s] marked as invalid by state validator. Reason code [%d]",
				block.Header.Number, txIndex, chdr.TxId, txsFilter.Flag(txIndex))
		}
	}
	v.moveCalsToUpdate(updates)
	//保存区块统计和当前统计
	blockStaticBuf, _ := proto.Marshal(curBlockStatistics)
	if blockStaticBuf != nil && vh != nil {
		updateStatic := &statedb.VersionedValue{blockStaticBuf, vh}

		mapStaticUpdates["curBlockStatistics"] = updateStatic
		mapStaticUpdates["BlockStatistics_"+string(curBlockStatistics.Number)] = updateStatic
	}

	logger.Infof("Validating a block with [%d] transactions with %d updates,%d calupdate,%d rlistupdate", len(block.Data.Data), updates.UpdateNum, updates.CalNum, updates.RListNum)

	//验证结束后，将
	block.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER] = txsFilter
	return updates, nil
}

func (v *Validator) calStatistics(mapUpdates map[string]*statedb.VersionedValue, mapTransStatic map[string]*common.TransactionStatistics, vh *version.Height, txId, chaincodeId, strFunc, vcode string) {
	if vh != nil {
		v.calStatistics1(mapUpdates, mapTransStatic, vh, txId, "", "", "")
		v.calStatistics1(mapUpdates, mapTransStatic, vh, txId, "", "", vcode)
		v.calStatistics1(mapUpdates, mapTransStatic, vh, txId, chaincodeId, "", "")
		v.calStatistics1(mapUpdates, mapTransStatic, vh, txId, chaincodeId, "", vcode)
		v.calStatistics1(mapUpdates, mapTransStatic, vh, txId, chaincodeId, strFunc, "")
		v.calStatistics1(mapUpdates, mapTransStatic, vh, txId, chaincodeId, strFunc, vcode)
	}

}

func (v *Validator) calStatistics1(mapUpdates map[string]*statedb.VersionedValue, mapTransStatic map[string]*common.TransactionStatistics, vh *version.Height, txId, chaincodeId, strFunc, vcode string) {
	transStaticKey := chaincodeId + "-" + strFunc + "-" + vcode
	transStatic := mapTransStatic[transStaticKey]
	if mapTransStatic[transStaticKey] == nil {
		transStatic = &common.TransactionStatistics{}
		mapTransStatic[transStaticKey] = transStatic
		curBlockStatistics.List = append(curBlockStatistics.List, transStatic)

	} else {
		updateTxStatic := &statedb.VersionedValue{[]byte(mapTransStatic[transStaticKey].PreTxId), vh}
		//logger.Infof("tx key %s pretxId %s ", txId+"-"+transStaticKey, mapTransStatic[transStaticKey].PreTxId)
		mapUpdates[txId+"-"+transStaticKey] = updateTxStatic
	}

	transStatic.PreTxId = txId
	transStatic.ChaincodeId = chaincodeId
	transStatic.Func = strFunc
	transStatic.ValidationCode = vcode

	mapTransStatic[transStaticKey].TxSum++
}

func addWriteSetToBatch(txRWSet *rwsetutil.TxRwSet, txHeight *version.Height, batch *statedb.UpdateBatch) {
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace
		for _, kvWrite := range nsRWSet.KvRwSet.Writes {
			if kvWrite.IsDelete {
				batch.Delete(ns, kvWrite.Key, txHeight)
			} else {
				vStr := string(kvWrite.Key)

				//ronger,王剑对值作增加计算
				if !strings.HasPrefix(vStr, "__CAL_") && !strings.HasPrefix(vStr, "__RLIST_") {
					batch.Put(ns, kvWrite.Key, kvWrite.Value, txHeight)
				}
			}
		}
	}
}

func (v *Validator) validateTx(txRWSet *rwsetutil.TxRwSet, updates *statedb.UpdateBatch) (peer.TxValidationCode, error) {
	for _, nsRWSet := range txRWSet.NsRwSets {
		ns := nsRWSet.NameSpace

		if valid, err := v.validateReadSet(ns, nsRWSet.KvRwSet.Reads, updates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_MVCC_READ_CONFLICT, nil
		}
		if valid, err := v.validateRangeQueries(ns, nsRWSet.KvRwSet.RangeQueriesInfo, updates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_PHANTOM_READ_CONFLICT, nil
		}

		//rongzer,验证交易前，先处理计算值
		bTime := time.Now().UnixNano() / 1000000

		if valid, err := v.calWriteSet(ns, nsRWSet.KvRwSet.Writes, updates); !valid || err != nil {
			if err != nil {
				return peer.TxValidationCode(-1), err
			}
			return peer.TxValidationCode_PHANTOM_READ_CONFLICT, nil
		}
		eTime := time.Now().UnixNano() / 1000000
		if eTime-bTime > 1 {
			logger.Infof("cal writeset use time %d", eTime-bTime)
		}
	}
	return peer.TxValidationCode_VALID, nil
}

func (v *Validator) calWriteSet(ns string, kvWrites []*kvrwset.KVWrite, updates *statedb.UpdateBatch) (bool, error) {

	updates.GetOrCreateNsUpdates(ns)
	updates.GetOrCreateNsCals(ns)
	updates.GetOrCreateNsRLists(ns)
	mapUpdates := updates.GetUpdates(ns)
	mapCals := updates.GetCals(ns)
	mapRLists := updates.GetRLists(ns)
	vh, _ := v.db.GetLatestSavePoint()

	for _, kvWrite := range kvWrites {
		vStr := string(kvWrite.Key)

		//ronger,王剑对值作增加计算
		if strings.HasPrefix(vStr, "__CAL_") {
			vStr = vStr[6:]
			calMethod := vStr[:3]

			calParams := strings.Split(vStr[4:], ",")

			lParam, _ := strconv.ParseInt(string(kvWrite.Value), 10, 64)

			existValue := mapCals[calParams[0]]
			//string到int64

			if existValue == nil {
				existValue = mapUpdates[calParams[0]]
				if existValue == nil {
					//读己有的值
					existValue, _ = v.db.GetState(ns, calParams[0])
				}
			}

			if existValue == nil {
				existValue = &statedb.VersionedValue{[]byte("0"), vh}
			}

			//计算
			lValue, err := strconv.ParseInt(string(existValue.Value), 10, 64)
			if err != nil {
				lValue = 0
			}

			if strings.EqualFold(calMethod, "ADD") {
				lValue = lValue + lParam
			}

			if strings.EqualFold(calMethod, "MUL") {
				lValue = lValue * lParam
			}

			logger.Debugf("cal kvWrite %s value method:%s param:%s old:%s result:%d",
				kvWrite.Key, calMethod, lParam, string(existValue.Value), lValue)

			//int64到string
			existValue.Value = []byte(strconv.FormatInt(lValue, 10))
			mapCals[calParams[0]] = existValue
			delete(mapUpdates, calParams[0])
		}

		//维护RList
		if strings.HasPrefix(vStr, "__RLIST_") {
			vStr = vStr[8:]
			lisMethod := vStr[:3]
			lisParams := strings.Split(vStr[4:], ",")

			rList := mapRLists[lisParams[0]]
			if rList == nil {
				//尝试用缓存处理
				if rListCache == nil {
					defaultExpiration, _ := time.ParseDuration("60s")
					gcInterval, _ := time.ParseDuration("3s")
					rListCache = cache.NewCache(defaultExpiration, gcInterval)
				}

				if v, found := rListCache.Get(lisParams[0]); found {
					//logger.Infof("rlist1 %s %p", lisParams[0], v)
					rList = v.(*statedb.RList)
					mapRLists[lisParams[0]] = rList
					// logger.Infof("rlist3 %s %p", lisParams[0], rList)
				}

			}
			if rList == nil {

				rList = statedb.NewRList(v.db, ns, lisParams[0])
				mapRLists[lisParams[0]] = rList
				//logger.Infof("rlist0 %s %p", lisParams[0], rList)
				expiration, _ := time.ParseDuration("30s")
				rListCache.Set(lisParams[0], rList, expiration)
			}

			if strings.EqualFold(lisMethod, "ADD") {
				lisValues := strings.Split(string(kvWrite.Value), ",")
				if len(lisValues) == 2 {
					b, error := strconv.Atoi(lisValues[0])
					if error != nil {
						rList.AddId(lisValues[1])
					} else {
						rList.AddIndexId(b, lisValues[1])
					}

				} else {
					rList.AddId(lisValues[0])
				}
			}

			if strings.EqualFold(lisMethod, "DEL") {
				rList.RemoveId(string(kvWrite.Value))
			}
		}

	}

	for _, kvWrite := range kvWrites {
		//处理值的变理
		valueStr := string(kvWrite.Value)

		if strings.Contains(valueStr, "##RCOUNT##") {
			replaceStr := valueStr

			for {
				bIndex := strings.Index(replaceStr, "##RCOUNT##")
				if bIndex < 0 {
					break
				}
				//logger.Infof("replaceStr %d %s", bIndex, replaceStr)

				replaceStr = replaceStr[bIndex+10:]
				eIndex := strings.Index(replaceStr, "##RCOUNT##")
				if eIndex <= 0 || eIndex > 64 {
					break
				}

				replaceVal := replaceStr[0:eIndex]

				//查找replaceVal对应的值，先从计算参数中找
				existValue := mapCals[replaceVal]
				if existValue == nil {
					//读己有的值
					existValue, _ = v.db.GetState(ns, replaceVal)
					if existValue == nil {
						existValue = &statedb.VersionedValue{[]byte("0"), vh}
					}
					mapCals[replaceVal] = existValue
				}

				//计算
				lValue, err := strconv.ParseInt(string(existValue.Value), 10, 64)
				if err != nil {
					lValue = 0
				}

				replaceStr = replaceStr[eIndex+10:]

				//替换valueStr中的值
				valueStr = strings.Replace(valueStr, "##RCOUNT##"+replaceVal+"##RCOUNT##", strconv.FormatInt(lValue, 10), -1)
				//logger.Infof("replaceStr Result %s", replaceStr)

				//logger.Infof("replaceValue %s", valueStr)

			}
			kvWrite.Value = []byte(valueStr)
		}

	}

	return true, nil
}

func (v *Validator) moveCalsToUpdate(updates *statedb.UpdateBatch) {
	namespaces := updates.GetCaledNamespaces()
	updates.CalNum = 0
	updates.RListNum = 0
	for _, ns := range namespaces {
		updates.GetOrCreateNsUpdates(ns)
		updates.GetOrCreateNsCals(ns)
		mapUpdates := updates.GetUpdates(ns)
		mapCals := updates.GetCals(ns)

		for k, vv := range mapCals {
			if mapUpdates[k] == nil {
				mapUpdates[k] = vv
				updates.CalNum++
			}
			delete(mapCals, k)
		}
	}

	//更新RList
	namespaces = updates.GetRListsNamespaces()
	for _, ns := range namespaces {
		vh, _ := v.db.GetLatestSavePoint()
		updates.GetOrCreateNsUpdates(ns)
		updates.GetOrCreateNsRLists(ns)
		mapUpdates := updates.GetUpdates(ns)
		mapRlists := updates.GetRLists(ns)

		for k, vv := range mapRlists {
			rList := vv
			//if rList.RListName == "__DEAL__ALL" {
			//	rList.Print(0)
			//}
			//logger.Infof("%s=========================================================================================", k)
			//    rList.Print(1)
			//logger.Infof("=========================================================================================")

			rList.SaveState()
			putStub := rList.GetPutStub()

			for j, value := range putStub {
				updateValue := &statedb.VersionedValue{value, vh}
				mapUpdates[j] = updateValue
				delete(putStub, j)
				updates.RListNum++
			}
			delete(mapRlists, k)
		}
	}

	updates.UpdateNum = updates.GetUpdateSize()
}

func (v *Validator) validateReadSet(ns string, kvReads []*kvrwset.KVRead, updates *statedb.UpdateBatch) (bool, error) {
	for _, kvRead := range kvReads {
		if valid, err := v.validateKVRead(ns, kvRead, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateKVRead performs mvcc check for a key read during transaction simulation.
// i.e., it checks whether a key/version combination is already updated in the statedb (by an already committed block)
// or in the updates (by a preceding valid transaction in the current block)
func (v *Validator) validateKVRead(ns string, kvRead *kvrwset.KVRead, updates *statedb.UpdateBatch) (bool, error) {
	if updates.Exists(ns, kvRead.Key) {
		return false, nil
	}
	versionedValue, err := v.db.GetState(ns, kvRead.Key)
	if err != nil {
		return false, nil
	}
	var committedVersion *version.Height
	if versionedValue != nil {
		committedVersion = versionedValue.Version
	}

	if !version.AreSame(committedVersion, rwsetutil.NewVersion(kvRead.Version)) {
		logger.Debugf("Version mismatch for key [%s:%s]. Committed version = [%s], Version in readSet [%s]",
			ns, kvRead.Key, committedVersion, kvRead.Version)
		return false, nil
	}
	return true, nil
}

func (v *Validator) validateRangeQueries(ns string, rangeQueriesInfo []*kvrwset.RangeQueryInfo, updates *statedb.UpdateBatch) (bool, error) {
	for _, rqi := range rangeQueriesInfo {
		if valid, err := v.validateRangeQuery(ns, rqi, updates); !valid || err != nil {
			return valid, err
		}
	}
	return true, nil
}

// validateRangeQuery performs a phatom read check i.e., it
// checks whether the results of the range query are still the same when executed on the
// statedb (latest state as of last committed block) + updates (prepared by the writes of preceding valid transactions
// in the current block and yet to be committed as part of group commit at the end of the validation of the block)
func (v *Validator) validateRangeQuery(ns string, rangeQueryInfo *kvrwset.RangeQueryInfo, updates *statedb.UpdateBatch) (bool, error) {
	logger.Debugf("validateRangeQuery: ns=%s, rangeQueryInfo=%s", ns, rangeQueryInfo)

	// If during simulation, the caller had not exhausted the iterator so
	// rangeQueryInfo.EndKey is not actual endKey given by the caller in the range query
	// but rather it is the last key seen by the caller and hence the combinedItr should include the endKey in the results.
	includeEndKey := !rangeQueryInfo.ItrExhausted

	combinedItr, err := newCombinedIterator(v.db, updates,
		ns, rangeQueryInfo.StartKey, rangeQueryInfo.EndKey, includeEndKey)
	if err != nil {
		return false, err
	}
	defer combinedItr.Close()
	var validator rangeQueryValidator
	if rangeQueryInfo.GetReadsMerkleHashes() != nil {
		logger.Debug(`Hashing results are present in the range query info hence, initiating hashing based validation`)
		validator = &rangeQueryHashValidator{}
	} else {
		logger.Debug(`Hashing results are not present in the range query info hence, initiating raw KVReads based validation`)
		validator = &rangeQueryResultsValidator{}
	}
	validator.init(rangeQueryInfo, combinedItr)
	return validator.validate()
}
