package com.rongzer.chaincode.base;

import java.util.Date;
import java.util.Iterator;
import java.util.List;

import org.apache.commons.codec.binary.Base64;
import org.apache.log4j.MDC;
import org.hyperledger.fabric.shim.ChaincodeBase;
import org.hyperledger.fabric.shim.ChaincodeStub;
import org.hyperledger.fabric.shim.ledger.KeyModification;
import org.hyperledger.fabric.shim.ledger.QueryResultsIterator;

import com.rongzer.chaincode.entity.HistoryEntity;
import com.rongzer.chaincode.entity.PageList;
import com.rongzer.chaincode.utils.StringUtil;

/**
 * RBC智能合约基类，实现run、query、getChaincodeID方法
 * @author Administrator
 *
 */
public abstract class RBCChaincodeBase extends ChaincodeBase {
	
	public abstract Response run(ChaincodeStub stub, String function, String[] args);
	public abstract Response query(ChaincodeStub stub, String function, String[] args);
    public abstract String getChaincodeID();

	@Override
	public Response init(ChaincodeStub stub) {
		final List<String> argList = stub.getStringArgs();
		final String function = argList.get(0);
		final String[] args = argList.stream().skip(1).toArray(String[]::new);

		return run(stub,function,args);
	}

	@Override
	public Response invoke(ChaincodeStub stub) {
		final List<String> argList = stub.getStringArgs();
		final String method = argList.get(0);
		final String function = argList.get(1);
		final String[] args = argList.stream().skip(2).toArray(String[]::new);
		Response response = null;
		
		if ("query".equals(method) || "query".equals(function) || function.toLowerCase().startsWith("query")|| function.toLowerCase().startsWith("get"))
		{
			if ("queryRBCCName".equals(function))//默认查询合约名称方法
			{
				response = newSuccessResponse(stub.getSecurityContext().getChainCodeName().getBytes());
			}else if ("queryStateHistory".equals(function))//默认查询状态值历史方法
			{
				if (args.length != 1)
				{
					newErrorResponse("query state history need 1 args");
				}
				
				QueryResultsIterator<KeyModification>  iter = stub.getHistoryForKey(args[0]);
				
				Iterator<KeyModification> iterator = iter.iterator();
				PageList<HistoryEntity> pageList = new PageList<HistoryEntity>();
				
				while (iterator.hasNext() ) {
					KeyModification keyModification = iterator.next();
					HistoryEntity historyEntity = new HistoryEntity();
					historyEntity.setTxId(keyModification.getTxId());					
					historyEntity.setTxTime(StringUtil.dateTimeToStr(new Date(keyModification.getTimestamp().toEpochMilli())));
					historyEntity.setValue(Base64.encodeBase64String(keyModification.getValue()));
					pageList.add(historyEntity);
					if (pageList.size()>100)
					{
						pageList.remove(0);
					}
				}
				
				response = newSuccessResponse(pageList.getBytes());
				
			}else
			{
				try{
					response = query(stub,function,args);
				}catch(Exception e)
				{
					response = newErrorResponse(e.toString());
				}
			}
			
		}else
		{
			try{

				response = run(stub,function,args);
				/*
				//rongzer 清除MDC线程对象池
				Hashtable hmdc = MDC.getContext();
				if (hmdc != null)
				{
					Iterator keys = hmdc.keySet().iterator();
					while (keys.hasNext()) {
						String key = (String) keys.next();
						Object value = hmdc.get(key);
						if(value instanceof RList)
						{
							try
							{
								RList rList = (RList)value;
								rList.saveState();
							}catch(Exception e1)
							{
								e1.printStackTrace();
							}
						}
					}
					
				}
				*/
			}catch(Exception e)
			{
				response = newErrorResponse(e.toString());
			}
			MDC.clear();

		}
		
		
		
		return response;
	}

}
