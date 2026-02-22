import { Brain, AlertCircle, CheckCircle, Fingerprint, Info, ShieldAlert } from 'lucide-react';
import { useState } from 'react';
import { type Analysis } from '../types/Analysis';

interface AnalysisProps {
  selectedNode: string | null;
}

export const AnalysisTab = ({ selectedNode }: AnalysisProps) => {
  const [data, setData] = useState<Analysis>({
    "is_malicious": true,
    "confidence": 0.85,
    "justification": "The package’s anomalous install-time behavior does not match its name/purpose (\"ssl-info\"). During installation it downloaded a third‑party binary (trufflehog) from GitHub (release-assets.githubusercontent.com), extracted and executed it. Trufflehog is a secrets-scanning tool; executing it at install time and reading numerous system files (root profiles, /etc/passwd, TLS cert stores, /root/.npm cache) is unusual and indicates automated local secret discovery. Most concerning: the node process made connections to 169.254.169.254:80 (the cloud instance metadata IP) — a common technique used to harvest cloud credentials/metadata. Combined indicators (fetch+execute external binary, scanning for secrets, probing instance metadata, reading root user files and profiles, and spawning shell commands) strongly suggest malicious or at least highly risky supply-chain behavior (credential/secret harvesting). Even if the intent were benign (e.g., self-scan), doing this during npm install with external binary execution and metadata access is unacceptable for a library. Recommend treating the package as malicious, blocking it, and rotating any potentially exposed credentials on affected hosts.",
    "indicators": [
      "Outbound connections to 169.254.169.254:80 (EC2 metadata service) from node process",
      "Download and execution of trufflehog binary from release-assets.githubusercontent.com and /tmp",
      "Execution of shell commands (/bin/sh) and spawned curl/tar/gzip to fetch and extract external binary",
      "Access to sensitive files and profiles: /root/.profile, /root/.bashrc, /etc/passwd, /etc/hosts and many TLS certificate stores",
      "Network/DNS activity to oss.trufflehog.org and github release asset IPs"
    ]
  });

  if (!data) {
    return <></>
  }

  const isMalicious = data.is_malicious;
  const confidencePercent = Math.round(data.confidence * 100);
  
  const statusColor = isMalicious ? "#ef4444" : "#22c55e";
  const statusBg = isMalicious ? "bg-red-500/10" : "bg-green-500/10";
  const statusBorder = isMalicious ? "border-red-500/50" : "border-green-500/50";

  return (
    <div className="h-full flex flex-col overflow-hidden text-gray-200" style={{ background: "#0a0a0a" }}>
      <div className={`p-6 border-b ${statusBg} ${statusBorder}`}>
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-4">
            <div className={`p-3 rounded-xl border ${statusBorder} bg-black/40`}>
              {isMalicious ? (
                <ShieldAlert className="w-5 h-5 text-red-500" />
              ) : (
                <CheckCircle className="w-5 h-5 text-green-500" />
              )}
            </div>
            <div>
              <h2 className="text-xl font-bold tracking-tight" style={{ color: statusColor }}>
                {isMalicious ? "THREAT DETECTED" : "NO THREAT IDENTIFIED"}
              </h2>
            </div>
          </div>
          <div className="text-right">
            <div className="text-xs text-gray-500 uppercase font-bold mb-1">Confidence Score</div>
            <div className="text-2xl font-mono font-bold" style={{ color: statusColor }}>
              {confidencePercent}%
            </div>
          </div>
        </div>
        
        <div className="mt-4 h-1.5 w-full bg-gray-800 rounded-full overflow-hidden">
          <div 
            className="h-full transition-all duration-1000" 
            style={{ 
              width: `${confidencePercent}%`, 
              backgroundColor: statusColor,
              boxShadow: `0 0 10px ${statusColor}`
            }} 
          />
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-6 space-y-8">
        <section>
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Brain className="w-5 h-5" />
            <h3 className="text-sm font-bold uppercase tracking-widest">Model Justification</h3>
          </div>
          <div className="p-4 rounded-lg bg-[#111827] border border-[#374151] relative overflow-hidden">
            <div className="absolute top-0 left-0 w-1 h-full" style={{ backgroundColor: statusColor }} />
            <p className="text-gray-300 leading-relaxed italic text-sm">
              "{data.justification}"
            </p>
          </div>
        </section>

        <section>
          <div className="flex items-center gap-2 mb-4 text-gray-400">
            <Fingerprint className="w-5 h-5" />
            <h3 className="text-sm font-bold uppercase tracking-widest">Threat Indicators</h3>
          </div>
          <div className="grid grid-cols-1 gap-3">
            {data.indicators.map((indicator, idx) => (
              <div 
                key={idx} 
                className="flex items-center gap-3 p-3 rounded bg-black border border-gray-800 hover:border-gray-600 transition-colors"
              >
                <AlertCircle className={`w-4 h-4 ${isMalicious ? 'text-red-400' : 'text-blue-400'}`} />
                <span className="font-mono text-xs text-gray-300">{indicator}</span>
              </div>
            ))}
            {data.indicators.length === 0 && (
              <div className="text-center py-8 border border-dashed border-gray-800 rounded">
                <p className="text-gray-600 text-sm">No specific indicators flagged.</p>
              </div>
            )}
          </div>
        </section>
      </div>

      <div className="p-4 bg-black border-t border-[#1f2937] flex items-center gap-2 text-[10px] text-gray-600 italic">
        <Info className="w-3 h-3" />
        <span>This analysis is generated by a large language model and should be used as a supporting tool alongside manual verification.</span>
      </div>
    </div>
  );
};