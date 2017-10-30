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

package rbccustomer

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hyperledger/fabric/common/flogging"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
)

var logger = flogging.MustGetLogger("lscc")

// rbccustomer Chaincode implementation
type RBCCustomer struct {
}

// Customer data entity
type CustomerEntity struct {
	TxId             string            `json:"txId"`             //交易Id
	TxTime           string            `json:"txTime"`           //交易时间
	IdKey            string            `json:"idKey"`            //对象存储的key
	CustomerId       string            `json:"customerId"`       //会员CustomerId(主键)
	CustomerNo       string            `json:"customerNo"`       //会员编号
	CustomerName     string            `json:"customerName"`     //会员名称
	CustomerType     string            `json:"customerType"`     //会员类型1:超级用户；2：审计用户;3：B端用户；4、C端用户,客户类型不能修改
	CustomerStatus   string            `json:"customerStatus"`   //会员状态:1正常；2锁定；3注销
	CustomerSignCert string            `json:"customerSignCert"` //会员公钥，用于会员验证
	CustomerAuth     string            `json:"customerAuth"`     //会员权限
	RegCustomerNo    string            `json:"regCustomerNo"`    //注册者
	RegTime          string            `json:"regTime"`          //注册时间
	Dict             map[string]string `json:"dict"`             //会员扩展信息Dict
}

// Init initializes the sample system chaincode by storing the key and value
// arguments passed in as parameters
func (t *RBCCustomer) Init(stub shim.ChaincodeStubInterface) pb.Response {
	//as system chaincodes do not take part in consensus and are part of the system,
	//best practice to do nothing (or very little) in Init.

	return shim.Success(nil)
}

// Invoke gets the supplied key and if it exists, updates the key with the newly
// supplied value.
func (t *RBCCustomer) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	args := stub.GetArgs()
	if len(args) < 3 {
		return shim.Error(fmt.Sprintf("Incorrect number of arguments%v. Expecting %d", args, len(args)))
	}

	f := string(args[1])

	switch f {
	case "queryRBCCName":
		return shim.Success([]byte("rbccustomer"))
	case "queryStateHistory":
		return shim.Success(stub.QueryStateHistory(string(args[2])))
	case "register":
		return t.register(stub, string(args[2]))
	case "modify":
		customerEntity, existCustomer, err := t.getCustomerInfo(stub, string(args[2]))
		if err != nil {
			resp := fmt.Sprintf("modify customer err %s", err)
			shim.Error(resp)
		}
		return t.modifyInfo(stub, customerEntity, existCustomer)
	case "modifyDict":
		customerEntity, existCustomer, err := t.getCustomerInfo(stub, string(args[2]))
		if err != nil {
			resp := fmt.Sprintf("modify customer err %s", err)
			shim.Error(resp)
		}
		customerEntity.CustomerName = ""
		customerEntity.CustomerType = ""
		customerEntity.CustomerStatus = ""
		customerEntity.CustomerSignCert = ""
		customerEntity.CustomerAuth = ""

		return t.modifyInfo(stub, customerEntity, existCustomer)
	case "queryOne":
		return t.queryOne(stub, string(args[2]))
	case "queryAll":
		return t.queryAll(stub, string(args[2]))
	case "testPerform":
		stub.PutState(string(args[2]), args[2])
		if strings.HasPrefix(string(args[2]), "RLIST") {
			stub.Add("testPerform", string(args[2]))
		}
		return shim.Success(nil)
	case "queryPerform":
		return t.queryPerform(stub, string(args[2]))
	default:
		jsonResp := fmt.Sprintf("function %s is not found", f)
		return shim.Error(jsonResp)
	}
}

// 查询交易访问者信息
// arguments passed in as parameters
func (t *RBCCustomer) getTxCustomerEntity(stub shim.ChaincodeStubInterface) *CustomerEntity {
	//取当前用户
	serializedIdentity := &msp.SerializedIdentity{}
	createBytes, _ := stub.GetCreator()
	err := proto.Unmarshal(createBytes, serializedIdentity)

	if err != nil {
		return nil
	}
	md5Ctx := md5.New()

	md5Ctx.Write([]byte(strings.TrimSpace(string(serializedIdentity.IdBytes))))
	md5Str := hex.EncodeToString(md5Ctx.Sum(nil))

	customerBuf, err := stub.GetState("__IDX_" + md5Str)
	if err != nil {
		return nil
	}
	customerId := string(customerBuf)
	if len(customerId) < 1 {
		return nil
	}
	existBytes, err := stub.GetState(customerId)
	if existBytes == nil {
		return nil
	}

	customerEntity := &CustomerEntity{}
	err = json.Unmarshal(existBytes, customerEntity)
	if err != nil {
		return nil
	}
	return customerEntity
}

// register customer
// arguments passed in as parameters
func (t *RBCCustomer) register(stub shim.ChaincodeStubInterface, params string) pb.Response {
	//获取注册者身份
	//查询原有多少会员
	nCustomerSize := stub.Size("customer")
	customerEntity := &CustomerEntity{}

	err := json.Unmarshal([]byte(params), customerEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("register param is not a json, %s", err))
	}

	//存在会员,就需要进行交易者身份判断
	if nCustomerSize > 0 {
		//获取注册者
		txCustomerEntity := t.getTxCustomerEntity(stub)
		if txCustomerEntity == nil {
			return shim.Error("tx user is not a customer")
		}
		customerType, err := strconv.Atoi(customerEntity.CustomerType)
		if err != nil {
			return shim.Error("customer type is invalid")
		}
		txCustomerType, err := strconv.Atoi(txCustomerEntity.CustomerType)
		if err != nil {
			return shim.Error("tx customer type is invalid")
		}
		//只有1，2，3类型才能注册用户
		if txCustomerType > 3 || txCustomerType < 1 {
			return shim.Error(fmt.Sprintf("tx user customer type is %s not in [1,2,3]", txCustomerEntity.CustomerType))
		}

		//1,2,3类型只能由1用户注册
		if customerType < 4 && txCustomerType != 1 {
			return shim.Error(fmt.Sprintf("customer type %s must regist by 1", customerEntity.CustomerType))
		}
		customerEntity.RegCustomerNo = txCustomerEntity.CustomerNo
	} else { //不存在会员,允许直接注册
		customerEntity.RegCustomerNo = "system"
	}

	//默认会员状态是正常
	customerEntity.CustomerStatus = "1"
	if customerEntity.CustomerId == "" || len(customerEntity.CustomerId) < 1 {
		return shim.Error(fmt.Sprintf("customerId is nil"))
	}

	//检查会员是否己经存在
	existBytes, err := stub.GetState(customerEntity.CustomerId)
	if existBytes != nil {
		return shim.Error(fmt.Sprintf("the customerId %s has exist", customerEntity.CustomerId))
	}

	existBytes, err = stub.GetState("__IDX_" + customerEntity.CustomerNo)
	if existBytes != nil {
		return shim.Error(fmt.Sprintf("the customerNo %s has exist", customerEntity.CustomerNo))
	}

	customerEntity.IdKey = customerEntity.CustomerId
	customerEntity.TxId = stub.GetTxID()
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	customerEntity.TxTime = txTimeStr
	customerEntity.RegTime = txTimeStr

	//保存会员信息bool
	customerBytes, err := json.Marshal(customerEntity)
	if err != nil {
		return shim.Error(fmt.Sprintf("convert customer entity to json err, %s", err))
	}
	stub.PutState(customerEntity.CustomerId, customerBytes)
	//维护索引
	stub.PutState("__IDX_"+customerEntity.CustomerId, []byte(customerEntity.CustomerId))
	stub.PutState("__IDX_"+customerEntity.CustomerNo, []byte(customerEntity.CustomerId))

	md5Ctx := md5.New()
	md5Ctx.Write([]byte(strings.TrimSpace(customerEntity.CustomerSignCert)))
	md5Str := hex.EncodeToString(md5Ctx.Sum(nil))
	stub.PutState("__IDX_"+md5Str, []byte(customerEntity.CustomerId))

	//维护列表
	stub.Add("customer", customerEntity.CustomerId)
	stub.Add("customer_"+customerEntity.CustomerType, customerEntity.CustomerId)
	stub.Add("customer_"+customerEntity.CustomerType+"_"+customerEntity.CustomerStatus, customerEntity.CustomerId)
	//rongzer,add by songxiang
	stub.Add("customer__"+customerEntity.CustomerStatus, customerEntity.CustomerId)
	return shim.Success([]byte("regist success"))
}

// arguments passed in as parameters
func (t *RBCCustomer) getCustomerInfo(stub shim.ChaincodeStubInterface, params string) (*CustomerEntity, *CustomerEntity, error) {
	customerEntity := &CustomerEntity{}

	err := json.Unmarshal([]byte(params), customerEntity)
	if err != nil {
		return nil, nil, fmt.Errorf("register param is not a json, %s", err)
	}

	//检查会员是否己经存在
	existBytes, err := stub.GetState(customerEntity.CustomerId)
	if existBytes == nil {
		return nil, nil, fmt.Errorf("the customerId %s not exist", customerEntity.CustomerId)
	}

	existCustomer := &CustomerEntity{}
	err = json.Unmarshal(existBytes, existCustomer)
	if err != nil {
		return nil, nil, fmt.Errorf("exist customer is not a json, %s", err)
	}

	return customerEntity, existCustomer, nil
}

// arguments passed in as parameters
func (t *RBCCustomer) getCustomerEntity(stub shim.ChaincodeStubInterface, customerQuery string) (*CustomerEntity, error) {
	customerBuf, err := stub.GetState("__IDX_" + customerQuery)
	if err != nil {
		return nil, err
	}
	customerId := string(customerBuf)

	//检查会员是否己经存在
	existBytes, err := stub.GetState(customerId)
	if existBytes == nil {
		return nil, fmt.Errorf("the customerId %s not exist", customerQuery)
	}

	existCustomer := &CustomerEntity{}
	err = json.Unmarshal(existBytes, existCustomer)
	if err != nil {
		return nil, fmt.Errorf("exist customer is not a json, %s", err)
	}

	return existCustomer, nil
}

// arguments passed in as parameters
func (t *RBCCustomer) modifyInfo(stub shim.ChaincodeStubInterface, customerEntity *CustomerEntity, existCustomer *CustomerEntity) pb.Response {

	//获取交易发起者
	txCustomerEntity := t.getTxCustomerEntity(stub)
	if txCustomerEntity == nil {
		return shim.Error("tx user is not a customer")
	}

	//修改会员名称、会员类型、会员状态、认证状态、会员证书必须由联合审批进行
	if len(customerEntity.CustomerName) > 0 || len(customerEntity.CustomerType) > 0 || len(customerEntity.CustomerStatus) > 0 || len(customerEntity.CustomerAuth) > 0 || len(customerEntity.CustomerSignCert) > 0 {
		//修改固由参数必须由超级用户进行
		if txCustomerEntity.CustomerType != "1" {
			return shim.Error("modify customer attribute must be 1")

		}
		chr, err := stub.GetChannelHeader()
		if err != nil {
			return shim.Error("get channel header has err")
		}
		chaincodeSpec := &pb.ChaincodeSpec{}
		err = proto.Unmarshal(chr.Extension, chaincodeSpec)
		if err != nil {
			return shim.Error("get chaincode spec has err")
		}

		//如果这些动作不是从rbcapproval过来
		if chaincodeSpec.ChaincodeId.Name != "rbcapproval" {
			return shim.Error("modify customer attribute must from rbcapproval")
		}
	}

	if len(customerEntity.CustomerName) > 0 {
		existCustomer.CustomerName = customerEntity.CustomerName
	}

	if len(customerEntity.CustomerAuth) > 0 {
		existCustomer.CustomerAuth = customerEntity.CustomerAuth
	}

	oldCustomerType := existCustomer.CustomerType
	oldCustomerStatus := existCustomer.CustomerStatus

	if len(customerEntity.CustomerType) > 0 {
		existCustomer.CustomerType = customerEntity.CustomerType
	}

	if len(customerEntity.CustomerStatus) > 0 {
		existCustomer.CustomerStatus = customerEntity.CustomerStatus
	}

	if len(customerEntity.CustomerSignCert) > 0 {
		md5Ctx := md5.New()
		md5Ctx.Write([]byte(strings.TrimSpace(existCustomer.CustomerSignCert)))
		md5Str := hex.EncodeToString(md5Ctx.Sum(nil))
		//删除索引
		stub.DelState("__IDX_" + md5Str)
		existCustomer.CustomerSignCert = customerEntity.CustomerSignCert
		md5Ctx1 := md5.New()
		md5Ctx1.Write([]byte(strings.TrimSpace(customerEntity.CustomerSignCert)))
		md5Str1 := hex.EncodeToString(md5Ctx1.Sum(nil))

		stub.PutState("__IDX_"+md5Str1, []byte(customerEntity.CustomerId))
	}

	if oldCustomerType != existCustomer.CustomerType {
		stub.Remove("customer_"+oldCustomerType, existCustomer.CustomerId)
		stub.Add("customer_"+existCustomer.CustomerType, existCustomer.CustomerId)
		stub.Remove("customer_"+oldCustomerType+"_"+oldCustomerStatus, existCustomer.CustomerId)
		stub.Add("customer_"+existCustomer.CustomerType+"_"+existCustomer.CustomerStatus, existCustomer.CustomerId)

	} else if oldCustomerStatus != existCustomer.CustomerStatus {
		stub.Remove("customer_"+oldCustomerType+"_"+oldCustomerStatus, existCustomer.CustomerId)
		stub.Add("customer_"+existCustomer.CustomerType+"_"+existCustomer.CustomerStatus, existCustomer.CustomerId)
		//rongzer,add by songxiang
		stub.Remove("customer__"+oldCustomerStatus, existCustomer.CustomerId)
		stub.Add("customer__"+existCustomer.CustomerStatus, existCustomer.CustomerId)
	}
	if customerEntity.Dict != nil {
		existCustomer.Dict = customerEntity.Dict
	}

	existCustomer.IdKey = customerEntity.CustomerId
	existCustomer.TxId = stub.GetTxID()
	txTime, _ := stub.GetTxTimestamp()
	tm := time.Unix(txTime.Seconds, 0)
	txTimeStr := tm.Format("2006-01-02 15:04:05")
	existCustomer.TxTime = txTimeStr

	//保存会员信息bool
	customerBytes, err := json.Marshal(existCustomer)
	if err != nil {
		return shim.Error(fmt.Sprintf("convert customer entity to json err, %s", err))
	}
	stub.PutState(customerEntity.CustomerId, customerBytes)

	return shim.Success([]byte("regist success"))
}

func (t *RBCCustomer) queryOne(stub shim.ChaincodeStubInterface, params string) pb.Response {
	//logger.Infof("INPUT params:" + params)

	customerBuf, err := stub.GetState("__IDX_" + params)
	if err != nil {
		return shim.Error(fmt.Sprintf("customer not found %s", err))
	}
	customerId := string(customerBuf)
	if len(customerId) < 1 {
		return shim.Success(nil)
	}
	existBytes, err := stub.GetState(customerId)
	if existBytes == nil {
		return shim.Success([]byte(""))
	}
	/*
		customerEntity := &CustomerEntity{}
		json.Unmarshal(existBytes, customerEntity)
		md5Ctx := md5.New()
		md5Ctx.Write([]byte(customerEntity.CustomerSignCert))
		md5Str := hex.EncodeToString(md5Ctx.Sum(nil))
		logger.Infof("MD5:" + md5Str)
		md5Ctx1 := md5.New()
		md5Ctx1.Write([]byte(strings.TrimSpace(customerEntity.CustomerSignCert)))
		md5Str1 := hex.EncodeToString(md5Ctx1.Sum(nil))
		logger.Infof("MD51:" + md5Str1)
	*/
	return shim.Success(existBytes)
}

func (t *RBCCustomer) queryAll(stub shim.ChaincodeStubInterface, params string) pb.Response {
	var m map[string]string
	err := json.Unmarshal([]byte(params), &m)
	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	// cpno 0:第一页 1：第二页
	cpnoStr := m["cpno"]
	cpno := 0
	if len(cpnoStr) > 0 {
		cpno, _ = strconv.Atoi(cpnoStr)
	}

	if cpno < 0 {
		cpno = 0
	}

	customerType := m["customerType"]
	customerStatus := m["customerStatus"]

	rListName := "customer"
	//rongzer,modify by songxiang
	if len(customerType) > 0 {
		rListName = rListName + "_" + customerType
		if len(customerStatus) > 0 {
			rListName = rListName + "_" + customerStatus
		}
	} else {
		if len(customerStatus) > 0 {
			rListName = rListName + "__" + customerStatus
		}
	}

	nSize := stub.Size(rListName)

	if cpno > nSize/20 {
		cpno = nSize / 20
	}

	bNo := cpno * 20
	eNo := (cpno + 1) * 20

	if eNo > nSize {
		eNo = nSize
	}

	listValue := make([]interface{}, 0)

	pageList := &shim.PageList{Cpno: strconv.Itoa(cpno), Rnum: strconv.Itoa(nSize), List: listValue}
	// 经典的循环条件初始化/条件判断/循环后条件变化
	for i := bNo; i < eNo; i++ {
		customerId := stub.Get(rListName, i)
		if len(customerId) > 0 {
			existBytes, _ := stub.GetState(customerId)
			if existBytes != nil {
				existCustomer := &CustomerEntity{}
				err = json.Unmarshal(existBytes, existCustomer)
				if err == nil {
					pageList.List = append(pageList.List, existCustomer)
				}
			}
		}
	}
	pageListBuf, err := json.Marshal(pageList)
	return shim.Success(pageListBuf)
}

func (t *RBCCustomer) queryPerform(stub shim.ChaincodeStubInterface, params string) pb.Response {
	var m map[string]string
	err := json.Unmarshal([]byte(params), &m)
	if err != nil {
		return shim.Error(fmt.Sprintf("query param is err: %s", err))
	}

	cpnoStr := m["cpno"]
	cpno := 0
	if len(cpnoStr) > 0 {
		cpno, _ = strconv.Atoi(cpnoStr)
	}

	if cpno < 0 {
		cpno = 0
	}
	rListName := "testPerform"
	nSize := stub.Size(rListName)

	if cpno > nSize/100 {
		cpno = nSize / 100
	}

	bNo := cpno * 100
	eNo := (cpno + 1) * 100

	if eNo > nSize {
		eNo = nSize
	}
	listValue := make([]interface{}, 0)

	pageList := &shim.PageList{Cpno: strconv.Itoa(cpno), Rnum: strconv.Itoa(nSize), List: listValue}
	// 经典的循环条件初始化/条件判断/循环后条件变化
	for i := bNo; i < eNo; i++ {
		customerId := stub.Get(rListName, nSize-i-1)

		existCustomer := &CustomerEntity{CustomerId: customerId}
		pageList.List = append(pageList.List, existCustomer)
	}
	pageListBuf, err := json.Marshal(pageList)
	return shim.Success(pageListBuf)
}
