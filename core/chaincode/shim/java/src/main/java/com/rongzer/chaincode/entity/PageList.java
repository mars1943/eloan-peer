package com.rongzer.chaincode.entity;

import java.util.ArrayList;

import net.sf.json.JSONArray;
import net.sf.json.JSONObject;

import com.rongzer.chaincode.utils.StringUtil;

public class PageList<V> extends ArrayList<V>{
	 public final static int PAGE_ROW = 20;

	int rnum = 0;
	int cpno = 0;
	
	public int getRnum() {
		return rnum;
	}

	public void setRnum(int rnum) {
		this.rnum = rnum;
	}

	public int getCpno() {
		return cpno;
	}

	public void setCpno(int cpno) {
		this.cpno = cpno;
	}

	public void fromJSON(JSONObject jData,Class<V> clazz)
	{
		if (jData == null || jData.isEmpty())
		{
			return;
		}
		rnum = StringUtil.toInt((String)jData.get("rnum"),0);
		cpno = StringUtil.toInt((String)jData.get("cpno"),0);
		JSONArray jList = jData.getJSONArray("list");
				
		if (jList != null && jList.size() >0)
		{
			for (int i=0;i<jList.size();i++)
			{
				
				try {
					BaseEntity baseEntity = (BaseEntity)clazz.newInstance();
					baseEntity.fromJSON(jList.getJSONObject(i));
					add((V)baseEntity);
				} catch (Exception e) {
					e.printStackTrace();
				} 
			}
		}	
	}
	
	public JSONObject toJSON()
	{
		JSONObject jData = new JSONObject();
		jData.put("rnum", ""+rnum);
		jData.put("cpno", ""+cpno);
		JSONArray jList = new JSONArray();
		cpno = StringUtil.toInt((String)jData.get("cpno"),0);
		for (int i=0;i<size();i++)
		{
			BaseEntity baseEntity = (BaseEntity)get(i);
			jList.add(baseEntity.toJSON());
		}
		jData.put("list", jList);
		
		return jData;
	}
	
	public byte[] getBytes()
	{
		JSONObject jData = toJSON();
		byte[] bReturn = new byte[0];
		try
		{
			bReturn = jData.toString().getBytes("UTF-8");
		}catch(Exception e)
		{
			
		}
		return bReturn;
	}
}
