import { Shield, HardDrive, Network, Terminal, Activity } from 'lucide-react';

export type BehavioralData = {
  collection: string;
  per_process: Record<string, {
    syscall_profile: Record<string, number>;
    file_access: Record<string, number>;
    executed_commands: Record<string, number>;
    network_activity: {
      ips: Record<string, number>;
      dns_records: Record<string, number>;
    };
  }>;
  count_processes: number;
  baseline_source: string;
  removed_processes: number;
  removed_files: number;
  removed_commands: number;
  removed_syscalls: number;
};

interface DataTabProps {
  selectedNode: string | null;
  data: BehavioralData | null;
}

export const DataTab = ({ selectedNode, data }: DataTabProps) => {
  const renderRecordList = (records: Record<string, number>, color = "#4ade80") => {
    const entries = Object.entries(records);
    if (entries.length === 0) return <div className="text-gray-500 italic">None detected</div>;

    return entries.map(([key, count], idx) => (
      <div key={idx} className="flex justify-between items-center gap-2">
        <span className="truncate" style={{ color }}>{key}</span>
        <span className="text-gray-500 text-[10px]">x{count}</span>
      </div>
    ));
  };

  if (!selectedNode) {
    return (
      <div className="h-full flex items-center justify-center" style={{ background: "#0a0a0a" }}>
        <p className="text-gray-500 text-sm">Select a package node to view behavioral data</p>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="h-full flex items-center justify-center" style={{ background: "#0a0a0a" }}>
        <p className="text-gray-500 text-sm">No behavioral data available for this package</p>
      </div>
    );
  }

  return (
    <div className="h-full flex flex-col overflow-hidden text-gray-200" style={{ background: "#0a0a0a" }}>
      <div className="p-4 border-b border-[#374151] bg-[#111827]">
        <div className="flex items-center justify-between mb-2">
          <h2 className="text-xl font-bold" style={{ color: "#22c55e" }}>
            {data.collection}
          </h2>
          <span className="text-xs px-2 py-1 rounded bg-[#22c55e20] text-[#22c55e] border border-[#22c55e]">
            {data.count_processes} ACTIVE PROCESSES
          </span>
        </div>
        <div className="flex flex-wrap gap-4 text-xs text-gray-400">
          <span>Baseline: <span className="text-gray-200">{data.baseline_source}</span></span>
          <span className="flex gap-2">
            <span className="text-red-400">-{data.removed_files} Files</span>
            <span className="text-red-400">-{data.removed_syscalls} Syscalls</span>
          </span>
        </div>
      </div>
      <div className="flex-1 overflow-y-auto p-4 space-y-8">
        {Object.entries(data.per_process).map(([processName, instances]) => (
          <section key={processName} className="border-l-2 border-[#374151] pl-4 py-1">
            <div className="flex items-center gap-2 mb-4">
              <Terminal className="w-5 h-5 text-blue-400" />
              <h3 className="text-lg font-mono font-bold text-blue-400 truncate">
                {processName}
              </h3>
              <span className="text-[10px] bg-blue-900/30 px-1 rounded text-blue-300">
                {Object.keys(instances).length} instance(s)
              </span>
            </div>
            <div className="space-y-6 mb-8 last:mb-0 bg-[#111827]/50 p-4 rounded-lg">
              <div>
                <div className="flex items-center gap-2 mb-2 text-sm font-semibold text-gray-300">
                  <HardDrive className="w-4 h-4" /> File Access
                </div>
                <div className="p-3 rounded bg-black border border-gray-800 font-mono text-xs space-y-1">
                  {renderRecordList(instances.file_access)}
                </div>
              </div>
              <div>
                <div className="flex items-center gap-2 mb-2 text-sm font-semibold text-gray-300">
                  <Network className="w-4 h-4" /> Network Activity
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div className="p-3 rounded bg-black border border-gray-800 font-mono text-xs">
                    <div className="text-[10px] uppercase text-gray-500 mb-1">IP Addresses</div>
                    {renderRecordList(instances.network_activity.ips, "#60a5fa")}
                  </div>
                  <div className="p-3 rounded bg-black border border-gray-800 font-mono text-xs">
                    <div className="text-[10px] uppercase text-gray-500 mb-1">DNS Records</div>
                    {renderRecordList(instances.network_activity.dns_records, "#818cf8")}
                  </div>
                </div>
              </div>
              <div>
                <div className="flex items-center gap-2 mb-2 text-sm font-semibold text-gray-300">
                  <Activity className="w-4 h-4" /> Syscall Profile
                </div>
                <div className="flex flex-wrap gap-2 p-3 rounded bg-black border border-gray-800 font-mono text-[10px]">
                  {Object.entries(instances.syscall_profile).map(([call, count]) => (
                    <span key={call} className="bg-gray-900 px-2 py-1 rounded border border-gray-700">
                      <span className="text-orange-400">{call}</span>
                      <span className="ml-1 text-gray-500">({count})</span>
                    </span>
                  ))}
                </div>
              </div>
            </div>
          </section>
        ))}
      </div>
      <div className="p-3 bg-[#111827] border-t border-[#374151] flex justify-between items-center text-[10px] uppercase tracking-wider text-gray-500">
        <div className="flex gap-4">
          <span>Commands Filtered: <span className="text-gray-300">{data.removed_commands}</span></span>
          <span>Processes Filtered: <span className="text-gray-300">{data.removed_processes}</span></span>
        </div>
        <Shield className="w-4 h-4 text-[#22c55e] opacity-50" />
      </div>
    </div>
  );
};
