import {
  AlertTriangle,
  CheckCircle,
  FileText,
  Network,
  HardDrive,
  Shield,
} from "lucide-react";
import { useState } from 'react';

interface AnalysisData {
  id: string;
  name: string;
  version: string;
  status: "safe" | "flagged" | "pending";
  riskLevel: "low" | "medium" | "high" | "critical";
  behaviors: {
    fileAccess: string[];
    networkCalls: string[];
    processSpawns: string[];
  };
  findings: {
    type: "info" | "warning" | "critical";
    message: string;
  }[];
  metadata: {
    author: string;
    downloads: string;
    lastPublished: string;
    license: string;
  };
}

interface AnalysisProps {
  selectedNode: string | null;
}

export function Analysis({ selectedNode }: AnalysisProps) {
  if (!selectedNode) return null;

  const [analysisData, setAnalysisData] = useState<AnalysisData>();

  const getStatusColor = (status: string) => {
    switch (status) {
      case "safe":
        return "#22c55e";
      case "flagged":
        return "#ef4444";
      default:
        return "#9ca3af";
    }
  };

  const getRiskColor = (risk: string) => {
    switch (risk) {
      case "low":
        return "#22c55e";
      case "medium":
        return "#eab308";
      case "high":
        return "#f97316";
      case "critical":
        return "#ef4444";
      default:
        return "#9ca3af";
    }
  };

  const getFindingIcon = (type: string) => {
    switch (type) {
      case "critical":
        return <AlertTriangle className="w-4 h-4" />;
      case "warning":
        return <AlertTriangle className="w-4 h-4" />;
      case "info":
        return <CheckCircle className="w-4 h-4" />;
      default:
        return null;
    }
  };

  const getFindingColor = (type: string) => {
    switch (type) {
      case "critical":
        return "#ef4444";
      case "warning":
        return "#f97316";
      case "info":
        return "#22c55e";
      default:
        return "#9ca3af";
    }
  };

  if (!analysisData) {
    return null;
  }

  return (
    <div
      className="h-full flex flex-col overflow-hidden"
      style={{
        background: "#0a0a0a",
      }}
    >
      <div
        className="p-4 border-b"
        style={{
          borderColor: "#374151",
          background: "#111827",
        }}
      >
        <div className="flex items-center gap-2 mb-2">
          <h2 className="text-xl" style={{ color: "#22c55e" }}>
            {analysisData.name}
          </h2>
          <span
            className="text-xs px-2 py-1 rounded"
            style={{
              background: getStatusColor(analysisData.status) + "20",
              color: getStatusColor(analysisData.status),
              border: `1px solid ${getStatusColor(analysisData.status)}`,
            }}
          >
            {analysisData.status.toUpperCase()}
          </span>
        </div>
        <div className="flex items-center gap-3 text-sm">
          <span style={{ color: "#9ca3af" }}>v{analysisData.version}</span>
          <span
            className="px-2 py-0.5 rounded text-xs"
            style={{
              background: getRiskColor(analysisData.riskLevel) + "20",
              color: getRiskColor(analysisData.riskLevel),
              border: `1px solid ${getRiskColor(analysisData.riskLevel)}`,
            }}
          >
            {analysisData.riskLevel.toUpperCase()} RISK
          </span>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto p-4 space-y-6">
        <section>
          <div className="flex items-center gap-2 mb-3">
            <Shield className="w-5 h-5" style={{ color: "#22c55e" }} />
            <h3 className="text-lg" style={{ color: "#22c55e" }}>
              Security Findings
            </h3>
          </div>
          <div className="space-y-2">
            {analysisData.findings.map((finding, idx) => (
              <div
                key={idx}
                className="p-3 rounded-lg border"
                style={{
                  background: "#1f2937",
                  borderColor: getFindingColor(finding.type),
                }}
              >
                <div className="flex items-start gap-2">
                  <span style={{ color: getFindingColor(finding.type) }}>
                    {getFindingIcon(finding.type)}
                  </span>
                  <p className="text-sm flex-1" style={{ color: "#e5e7eb" }}>
                    {finding.message}
                  </p>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section>
          <div className="flex items-center gap-2 mb-3">
            <HardDrive className="w-5 h-5" style={{ color: "#22c55e" }} />
            <h3 className="text-lg" style={{ color: "#22c55e" }}>
              File System Access
            </h3>
          </div>
          <div
            className="p-3 rounded-lg font-mono text-xs space-y-1"
            style={{
              background: "#000000",
              border: `1px solid #374151`,
            }}
          >
            {analysisData.behaviors.fileAccess.length > 0 ? (
              analysisData.behaviors.fileAccess.map((file, idx) => (
                <div
                  key={idx}
                  style={{
                    color: file.includes("⚠️") ? "#ef4444" : "#4ade80",
                  }}
                >
                  {file}
                </div>
              ))
            ) : (
              <div style={{ color: "#9ca3af" }}>No file access detected</div>
            )}
          </div>
        </section>

        <section>
          <div className="flex items-center gap-2 mb-3">
            <Network className="w-5 h-5" style={{ color: "#22c55e" }} />
            <h3 className="text-lg" style={{ color: "#22c55e" }}>
              Network Activity
            </h3>
          </div>
          <div
            className="p-3 rounded-lg font-mono text-xs space-y-1"
            style={{
              background: "#000000",
              border: `1px solid #374151`,
            }}
          >
            {analysisData.behaviors.networkCalls.length > 0 ? (
              analysisData.behaviors.networkCalls.map((call, idx) => (
                <div
                  key={idx}
                  style={{
                    color: call.includes("⚠️") ? "#ef4444" : "#4ade80",
                  }}
                >
                  {call}
                </div>
              ))
            ) : (
              <div style={{ color: "#9ca3af" }}>No network calls detected</div>
            )}
          </div>
        </section>

        {analysisData.behaviors.processSpawns.length > 0 && (
          <section>
            <div className="flex items-center gap-2 mb-3">
              <AlertTriangle className="w-5 h-5" style={{ color: "#ef4444" }} />
              <h3 className="text-lg" style={{ color: "#ef4444" }}>
                Process Spawning
              </h3>
            </div>
            <div
              className="p-3 rounded-lg font-mono text-xs space-y-1"
              style={{
                background: "#000000",
                border: "1px solid #ef4444",
              }}
            >
              {analysisData.behaviors.processSpawns.map((process, idx) => (
                <div key={idx} style={{ color: "#ef4444" }}>
                  {process}
                </div>
              ))}
            </div>
          </section>
        )}

        <section>
          <div className="flex items-center gap-2 mb-3">
            <FileText className="w-5 h-5" style={{ color: "#22c55e" }} />
            <h3 className="text-lg" style={{ color: "#22c55e" }}>
              Package Metadata
            </h3>
          </div>
          <div
            className="p-3 rounded-lg space-y-2 text-sm"
            style={{
              background: "#1f2937",
              border: `1px solid #374151`,
            }}
          >
            <div className="flex justify-between">
              <span style={{ color: "#9ca3af" }}>Author:</span>
              <span style={{ color: "#e5e7eb" }}>{analysisData.metadata.author}</span>
            </div>
            <div className="flex justify-between">
              <span style={{ color: "#9ca3af" }}>Downloads:</span>
              <span style={{ color: "#e5e7eb" }}>
                {analysisData.metadata.downloads}
              </span>
            </div>
            <div className="flex justify-between">
              <span style={{ color: "#9ca3af" }}>Last Published:</span>
              <span style={{ color: "#e5e7eb" }}>
                {analysisData.metadata.lastPublished}
              </span>
            </div>
            <div className="flex justify-between">
              <span style={{ color: "#9ca3af" }}>License:</span>
              <span style={{ color: "#e5e7eb" }}>{analysisData.metadata.license}</span>
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}
