import React, { useEffect, useState } from 'react'
import { MetricData, DataSeries } from './types'
import { CHART_CONFIG } from './chart-manifest'
import LineChart from './components/LineChart'
import BarChart from './components/BarChart'

function App() {
  const [data, setData] = useState<DataSeries[]>([])

  useEffect(() => {
    async function loadMetrics(url: string) {
      try {
        const response = await fetch(url)
        const jsonData = await response.json()
        return jsonData
      } catch (error) {
        console.error('Error loading metrics:', error)
      }
    }

    Promise.all([loadMetrics('/output/geth-validator/metrics.json'), loadMetrics('/output/geth-sequencer/metrics.json')])
      .then(([validatorMetrics, sequencerMetrics]) => {
        setData([
          {
            name: 'Validator',
            data: validatorMetrics
          },
          {
            name: 'Sequencer',
            data: sequencerMetrics
          }
        ])
      })
  }, [])

  if (data.length === 0) {
    return <div>Loading...</div>
  }

  return (
    <div className="container">
      <h1>Base Bench Metrics</h1>
      <div className="charts-container">
        {Object.entries(CHART_CONFIG).map(([metricKey, config]) => {
          const chartProps = {
            series: data,
            metricKey,
            title: config.title,
            description: config.description,
            unit: config.unit
          }

          return (
            <div key={metricKey} className="chart-container">
              {config.type === 'line' ? (
                <LineChart {...chartProps} />
              ) : (
                <BarChart {...chartProps} />
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

export default App 