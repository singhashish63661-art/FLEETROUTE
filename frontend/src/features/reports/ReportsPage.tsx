import React, { useEffect, useState } from 'react'
import { LineChart, Line, AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import api from '../../shared/api/client'

export function ReportsPage() {
  const [fuelData, setFuelData] = useState([])
  const [driverData, setDriverData] = useState([])

  useEffect(() => {
    // Fetch fuel data
    api.get('/reports/fuel').then((res) => {
      setFuelData(res.data.data)
    })
    
    // Fetch driver score data
    api.get('/reports/driver-behavior').then((res) => {
      setDriverData(res.data.data)
    })
  }, [])

  return (
    <div style={{ padding: '24px', color: '#fff', height: '100vh', overflowY: 'auto' }}>
      <h1>Analytics & Reports</h1>
      
      <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px', marginTop: '24px' }}>
        {/* Fuel Consumption Chart */}
        <div style={{ backgroundColor: '#1e1e1e', padding: '16px', borderRadius: '8px' }}>
          <h3>Fuel Usage Over Time</h3>
          <ResponsiveContainer width="100%" height={300}>
            <AreaChart data={fuelData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis dataKey="date" stroke="#888" />
              <YAxis stroke="#888" />
              <Tooltip contentStyle={{ backgroundColor: '#111', border: 'none' }} />
              <Area type="monotone" dataKey="liters" stroke="#10b981" fill="#10b981" fillOpacity={0.2} />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Driver Behavior Chart */}
        <div style={{ backgroundColor: '#1e1e1e', padding: '16px', borderRadius: '8px' }}>
          <h3>Driver Scores</h3>
          <ResponsiveContainer width="100%" height={300}>
            <LineChart data={driverData}>
              <CartesianGrid strokeDasharray="3 3" stroke="#333" />
              <XAxis dataKey="driver" stroke="#888" />
              <YAxis domain={[0, 100]} stroke="#888" />
              <Tooltip contentStyle={{ backgroundColor: '#111', border: 'none' }} />
              <Line type="monotone" dataKey="score" stroke="#3b82f6" strokeWidth={3} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </div>
    </div>
  )
}
