package com.rongzer.chaincode.utils;

import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.ObjectInputStream;
import java.io.ObjectOutputStream;
import java.util.ArrayList;
import java.util.Date;
import java.util.List;

import net.sf.json.JSONObject;

import org.hyperledger.fabric.shim.ChaincodeStub;

import com.google.protobuf.ByteString;
import com.rongzer.chaincode.entity.CustomerEntity;
import com.rongzer.chaincode.entity.PageList;

public class ChainCodeUtils {


	
	/**
	 * 获取交易时间
	 * @param stub
	 * @return
	 */
	public static String getTxTime(ChaincodeStub stub)
	{
		String strReturn = "";
		long lTime = stub.getSecurityContext().getTxTimestamp().getSeconds()*1000;
		strReturn = StringUtil.dateTimeToStr(new Date(lTime));
		return strReturn;
	}
	
	/**
	 * 根据条件查询用户
	 * @param stub
	 * @param jData
	 * @return
	 */
	public static CustomerEntity queryCustomer(ChaincodeStub stub, String param)
	{
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("queryOne"));
		lisArgs.add(ByteString.copyFromUtf8(param));
		String strReturn = stub.queryChaincode("rbccustomer","query", lisArgs);
		
		JSONObject jReturn = JSONUtil.getJSONObjectFromStr(strReturn);
		if(jReturn == null || jReturn.isEmpty()){
			return null;
		}
		CustomerEntity customerBean = new CustomerEntity();
		customerBean.fromJSON(jReturn);
		
		return customerBean;
	}
	
	/**
	 * 根据用户ID查询用户
	 * @param stub
	 * @param customerId
	 * @return
	 */
	public static CustomerEntity queryCustomerByID(ChaincodeStub stub, String customerId)
	{

		return queryCustomer(stub,customerId);
	}
	
	/**
	 * 根据用户NO查询用户
	 * @param stub
	 * @param customerNo
	 * @return
	 */
	public static CustomerEntity queryCustomerByNo(ChaincodeStub stub, String customerNo)
	{
		return queryCustomer(stub,customerNo);
	}
	
	/**
	 * 根据签名公钥查询用户
	 * @param stub
	 * @param searchSign
	 * @return
	 */
	public static CustomerEntity queryExecCustomer(ChaincodeStub stub)
	{

		return queryCustomer(stub, StringUtil.MD5(stub.getSecurityContext().getCallerCert().toStringUtf8().trim()));
	}
	
	
	/**
	 * 根据签名公钥查询用户
	 * @param stub
	 * @param searchSign
	 * @return
	 */
	public static CustomerEntity queryCustomerByCert(ChaincodeStub stub,String searchSign)
	{
		return queryCustomer(stub,StringUtil.MD5(searchSign));
	}
	
	/**
	 * 分页查询所有用户
	 * @param stub
	 * @param nPageNum
	 * @return
	 */
	public static PageList<CustomerEntity> queryAllCustomer(ChaincodeStub stub,String customerType,String customerStatus,int cpno)
	{

		JSONObject jData = new JSONObject();
		jData.put("cpno", ""+cpno);
		if (StringUtil.isNotEmpty(customerType))
		{
			jData.put("customerType", ""+customerType);
		}
		if (StringUtil.isNotEmpty(customerType))
		{
			jData.put("customerStatus", ""+customerStatus);
		}
		List<ByteString> lisArgs = new ArrayList<ByteString>();
		lisArgs.add(ByteString.copyFromUtf8("queryAll"));
		lisArgs.add(ByteString.copyFromUtf8(jData.toString()));
		String strReturn  = stub.invokeChaincode("rbccustomer","query", lisArgs);
		
		JSONObject jReturn = JSONUtil.getJSONObjectFromStr(strReturn);
		
		PageList<CustomerEntity> listCustomer = new PageList<CustomerEntity>();
		
		listCustomer.fromJSON(jReturn, CustomerEntity.class);

		return listCustomer;
	}	
	
	/**
	 * BaseStatic转ByteString
	 * @return
	 */
	public static ByteString toByteString(Object object)
	{
		
		ByteString byteString = null;
		ByteArrayOutputStream  byteOut = null;
		ObjectOutputStream out = null;
		try 
		{
			byteOut = new ByteArrayOutputStream();
			out = new ObjectOutputStream(byteOut);
			out.writeObject(object);
			byteString = ByteString.copyFrom(byteOut.toByteArray());			
			
		} catch (Exception e) {
			e.printStackTrace();
		}finally
		{
			try
			{
				if (byteOut != null)
				{
					byteOut.flush();
					byteOut.close();
					byteOut = null;
				}
				
				if (out != null)
				{
					out.flush();
					out.close();
					out = null;
				}
			}catch(Exception e)
			{
				
			}
			
		}
		
		return byteString;
		
	}
	

	
	/**
	 * 对内存的克隆处理
	 * @param object
	 * @return
	 */
	public static Object toObject(ByteString byteString)
	{
		
		Object objReturn = null;
		ObjectOutputStream out = null;
		try 
		{
			if (byteString == null)
			{
				return null;
			}
			
			byte[] b= byteString.toByteArray();
			if (byteString == null || b.length<1)
			{
				return null;
			}
			
			ByteArrayInputStream byteIn = new ByteArrayInputStream(b);
			ObjectInputStream in =new ObjectInputStream(byteIn);
			objReturn =in.readObject();
			
			byteIn.close();
			byteIn = null;
			in.close();
			in = null;
			
		} catch (Exception e) {
			e.printStackTrace();
		}finally
		{
			try
			{
				if (out != null)
				{
					out.flush();
					out.close();
					out = null;
				}
			}catch(Exception e)
			{
				
			}
			
		}
		
		return objReturn;
	}
}
