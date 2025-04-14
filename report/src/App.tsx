import { useEffect, useState } from 'react'
import { BenchmarkRuns, DataSeries } from './types'
import { CHART_CONFIG } from './chart-manifest'
import LineChart from './components/LineChart'
import BarChart from './components/BarChart'
import ChartSelector from './ChartSelector'

function App() {
  const [benchmarkRuns, setBenchmarkRuns] = useState<BenchmarkRuns | null>(null)

  useEffect(() => {
    async function loadMetrics() {
      try {
        const response = await fetch('/output/test_metadata.json')
        const jsonData = await response.json()
        setBenchmarkRuns(jsonData)
      } catch (error) {
        console.error('Error loading metrics:', error)
      }
    }
    loadMetrics()
  }, [])

  const fetchMetrics = async (outputDir: string, nodeType: string) => {
    const response = await fetch(`/output/${outputDir}/metrics-${nodeType}.json`)
    return await response.json()
  }

  const fetchResult = async (outputDir: string, nodeType: string) => {
    const response = await fetch(`/output/${outputDir}/result-${nodeType}.json`)
    return await response.json()
  }

  const getLogsDownloadGz = (outputDir: string, nodeType: string) => {
    return `/output/${outputDir}/logs-${nodeType}.tar.gz`
  }

  if (!benchmarkRuns || benchmarkRuns.runs.length === 0) {
    return <div>Loading...</div>
  }

  return (
    <div className="container">
      <h1>Base Bench Metrics</h1>
      <ChartSelector benchmarkRuns={benchmarkRuns} fetchMetrics={fetchMetrics} fetchResult={fetchResult} getLogsDownloadGz={getLogsDownloadGz} />
    </div>
  )
}

export default App 