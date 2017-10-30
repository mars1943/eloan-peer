package com.rongzer.chaincode.entity;

import org.apache.log4j.MDC;
import org.hyperledger.fabric.shim.ChaincodeStub;

import com.rongzer.chaincode.utils.StringUtil;

/**
 * 自动带时间顺序的列表
 * @author Administrator
 *
 */
public class RList {
	//数据操作对象
	private ChaincodeStub stub = null;
	
	//操作key
	private String rListName = "";
	
	private final static String prefix = "__RLIST_";
	private int seq = -1;

	
	public static RList getInstance(ChaincodeStub stub,String rListName)	
	{
		RList rList = null;
		//增加线程级缓存，可以避免多次初始化
		if (MDC.get(prefix+rListName) != null )
		{
			rList = (RList)MDC.get(prefix+rListName);
		}else
		{
			rList = new RList(stub,rListName);
			MDC.put(prefix+rListName, rList);
		}
		
		return rList;
	}
	
	private RList(ChaincodeStub stub,String rListName)
	{
		this.rListName = rListName;
		this.stub = stub;
	}
	
	private int getSeq()
	{
		return seq++;
	}
	
	
	/**
	 * 增加一个数据
	 * @param id
	 */
	public void add(String id)
	{
		stub.putStringState(prefix+"ADD:"+rListName+","+getSeq(),id);
	}
	
	/**
	 * 在某一个位置增加一个数据
	 * @param nIndex
	 * @param id
	 */
	public void add(int nIndex,String id)
	{
		stub.putStringState(prefix+"ADD:"+rListName+","+getSeq(),nIndex+","+id);
	}
	
	
	/**
	 * 移除一个数据
	 * @param id
	 */
	public void remove(String id)
	{
		stub.putStringState(prefix+"DEL:"+rListName+","+getSeq(),id);
	}
	
	/**
	 * 总长度
	 * @param id
	 */
	public int size()
	{
		return StringUtil.toInt(stub.getStringState(prefix+"LEN:"+rListName),0);
	}
	
	/**
	 * 总长度
	 * @param id
	 */
	public String get(int nIndex)
	{
		return stub.getStringState(prefix+"GET:"+rListName+","+nIndex);
	}
	
	/**
	 * 总长度
	 * @param id
	 */
	public int indexOf(String id)
	{
		return StringUtil.toInt(stub.getStringState(prefix+"IDX:"+rListName+","+id));
	}
}
