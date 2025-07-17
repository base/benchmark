import { BenchmarkRun } from "../types";

interface ProvidedProps {
  benchmarkRuns: BenchmarkRun[];
}

const BenchmarkRunDetails = ({ benchmarkRuns }: ProvidedProps) => {
  if (benchmarkRuns.length === 0) {
    return null;
  }

  return (
    <div className="flex flex-col gap-4 mb-8">
      <h1 className="text-2xl font-bold">{benchmarkRuns[0].testName}</h1>
      <p className="text-sm text-slate-500">
        {new Intl.DateTimeFormat("en-US", {
          dateStyle: "long",
          timeStyle: "short",
        }).format(new Date(benchmarkRuns[0].createdAt))}
      </p>
      <p className="text-sm text-slate-700 max-w-2xl">
        {benchmarkRuns[0].testDescription}
      </p>
    </div>
  );
};

export default BenchmarkRunDetails;
