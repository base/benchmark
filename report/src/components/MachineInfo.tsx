import { MachineInfo as MachineInfoType } from "../types";

interface MachineInfoProps {
  machineInfo?: MachineInfoType;
}

const getProviderUrl = (
  provider: string,
  machineType: string,
): string | null => {
  if (provider === "aws") {
    // Extract instance family from machine type (e.g., i4i from i4i.32xlarge)
    const instanceFamily = machineType.split(".")[0];
    return `https://aws.amazon.com/ec2/instance-types/${instanceFamily}/#Product-Details`;
  } else if (provider === "gcp") {
    const instanceFamily = machineType.split("-")[0];
    return `https://cloud.google.com/compute/docs/storage-optimized-machines#${instanceFamily}_machine_types`;
  }
  return null;
};

const MachineInfo = ({ machineInfo }: MachineInfoProps) => {
  if (
    !machineInfo ||
    (!machineInfo.type &&
      !machineInfo.provider &&
      !machineInfo.region &&
      !machineInfo.fileSystem)
  ) {
    return (
      <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
        <h3 className="text-sm font-semibold text-slate-700 mb-2">
          Machine Information
        </h3>
        <p className="text-sm text-slate-500">
          Machine information not available for this benchmark run.
        </p>
      </div>
    );
  }

  const providerUrl =
    machineInfo.provider && machineInfo.type
      ? getProviderUrl(machineInfo.provider, machineInfo.type)
      : null;

  return (
    <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
      <h3 className="text-sm font-semibold text-slate-700 mb-3">
        Machine Information
      </h3>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
        {machineInfo.type && (
          <div>
            <span className="text-slate-500 block">Type</span>
            {providerUrl ? (
              <a
                href={providerUrl}
                target="_blank"
                rel="noopener noreferrer"
                className="text-blue-600 hover:text-blue-800 hover:underline font-medium"
              >
                {machineInfo.type}
              </a>
            ) : (
              <span className="text-slate-900 font-medium">
                {machineInfo.type}
              </span>
            )}
          </div>
        )}
        {machineInfo.provider && (
          <div>
            <span className="text-slate-500 block">Provider</span>
            <span className="text-slate-900 font-medium uppercase">
              {machineInfo.provider}
            </span>
          </div>
        )}
        {machineInfo.region && (
          <div>
            <span className="text-slate-500 block">Region</span>
            <span className="text-slate-900 font-medium">
              {machineInfo.region}
            </span>
          </div>
        )}
        {machineInfo.fileSystem && (
          <div>
            <span className="text-slate-500 block">File System</span>
            <span className="text-slate-900 font-medium">
              {machineInfo.fileSystem}
            </span>
          </div>
        )}
      </div>
    </div>
  );
};

export default MachineInfo;
