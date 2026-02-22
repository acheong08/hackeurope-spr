
export type Data = {
  collection: string;
  per_process: Record<string, {
    syscall_profile: Record<string, number>;
    file_access: Record<string, number>;
    executed_commands: Record<string, number>;
    network_activity: {
      ips: Record<string, number>;
      dns_records: Record<string, number>
    }
  }>;
  count_processes: number;
  baseline_source: string;
  removed_processes: number;
  removed_files: number;
  removed_commands: number;
  removed_syscalls: number;
};