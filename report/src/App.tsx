import { useEffect, useMemo, useState } from 'react'
import { BenchmarkRuns } from './types'
import ChartSelector, { DataFileRequest } from './components/ChartSelector'
import ChartGrid from './components/ChartGrid'
import useMultipleDataSeries from './utils/useDataSeries'

function App() {
  const [benchmarkRuns, setBenchmarkRuns] = useState<BenchmarkRuns | null>(null)
  const [dataQuery, setDataQuery] = useState<DataFileRequest[]>([])

  const dataQueryKey = useMemo(() => {
    return dataQuery.map((query) => [query.outputDir, query.role] as [string, string])
  }
  , [dataQuery])

  const { data: dataPerFile, isLoading } = useMultipleDataSeries(dataQueryKey)
  const data = useMemo(() => {
    if (!dataPerFile) {
      return dataPerFile
    }

    return dataPerFile.map((data, index) => {
      const { name } = dataQuery[index]
      return {
        name,
        data,
      }
    })
}, [dataPerFile, dataQuery]);

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


  if (!benchmarkRuns || benchmarkRuns.runs.length === 0) {
    return <div>Loading...</div>
  }

  return (
    <div className="container">
      <h1>Base Bench Metrics</h1>
      <ChartSelector onChangeDataQuery={setDataQuery} benchmarkRuns={benchmarkRuns} />
      {isLoading ? "Loading..." : <ChartGrid data={data ?? []} />}
    </div>
  )
}

export default App 