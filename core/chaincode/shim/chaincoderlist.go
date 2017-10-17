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

// Package shim provides APIs for the chaincode to access its state
//rongzer,王剑增加
package shim

import (
	"encoding/base64"
	"encoding/json"
	"strconv"
	"time"
)

// Customer data entity
type PageList struct {
	Cpno string        `json:"cpno"` //当前页数,
	Rnum string        `json:"rnum"` //总记录数
	List []interface{} `json:"list"` //列表对象
}

// HistoryEntity
type HistoryEntity struct {
	TxId   string `json:"txId"`   //交易id,
	TxTime string `json:"txTime"` //交易时间
	Value  string `json:"value"`  //值
}

const prefix = "__RLIST_"

func (stub *ChaincodeStub) Add(rlistName string, id string) {
	stub.seq = stub.seq + 1
	stub.PutState(prefix+"ADD:"+rlistName+","+strconv.Itoa(stub.seq), []byte(id))
}

func (stub *ChaincodeStub) AddByIndex(rlistName string, nIndex int, id string) {
	stub.seq = stub.seq + 1
	stub.PutState(prefix+"ADD:"+rlistName+","+strconv.Itoa(stub.seq), []byte(strconv.Itoa(nIndex)+","+id))
}

func (stub *ChaincodeStub) Remove(rListName string, id string) {
	stub.seq = stub.seq + 1
	stub.PutState(prefix+"DEL:"+rListName+","+strconv.Itoa(stub.seq), []byte(id))
}

func (stub *ChaincodeStub) Size(rListName string) int {
	sizeBytes, err := stub.GetState(prefix + "LEN:" + rListName)
	if err != nil {
		return 0
	}
	nSize, err := strconv.Atoi(string(sizeBytes))
	if err != nil {
		nSize = 0
	}
	return nSize
}

func (stub *ChaincodeStub) Get(rListName string, nIndex int) string {
	sizeBytes, err := stub.GetState(prefix + "GET:" + rListName + "," + strconv.Itoa(nIndex))
	if err != nil {
		return ""
	}
	return string(sizeBytes)
}

func (stub *ChaincodeStub) IndexOf(rListName string, id string) int {
	returnBytes, err := stub.GetState(prefix + "IDX:" + rListName + "," + id)
	if err != nil {
		return -1
	}
	nReturn, err := strconv.Atoi(string(returnBytes))
	if err != nil {
		nReturn = -1
	}
	return nReturn
}

//取状态值的变更历史
func (stub *ChaincodeStub) QueryStateHistory(idKey string) []byte {
	if len(idKey) < 1 {
		return []byte("")
	}
	keysIter, err := stub.GetHistoryForKey(idKey)
	if err != nil {
		return []byte("")
	}
	defer keysIter.Close()

	listValue := make([]interface{}, 0)
	pageList := &PageList{Cpno: "0", List: listValue}

	nSize := 0
	for keysIter.HasNext() {
		response, iterErr := keysIter.Next()

		if iterErr != nil {
			return []byte("")
		}
		tm := time.Unix(response.Timestamp.Seconds, 0)
		txTimeStr := tm.Format("2006-01-02 15:04:05")
		historyEntity := &HistoryEntity{TxId: response.TxId, TxTime: txTimeStr}
		historyEntity.Value = base64.StdEncoding.EncodeToString(response.Value)
		pageList.List = append(pageList.List, historyEntity)

		nSize++
	}
	pageList.Rnum = strconv.Itoa(nSize)
	pageListBuf, _ := json.Marshal(pageList)
	return pageListBuf
}
