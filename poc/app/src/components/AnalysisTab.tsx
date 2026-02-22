import { Shield, ShieldAlert, AlertTriangle } from 'lucide-react';

export type SecurityAssessment = {
  is_malicious: boolean;
  confidence: number;
  justification: string;
  indicators?: string[];
};

interface AnalysisTabProps {
  selectedNode: string | null;
  assessment: SecurityAssessment | null;
}

export function AnalysisTab({ selectedNode, assessment }: AnalysisTabProps) {
  if (!selectedNode) {
    return (
      <div className="h-full flex items-center justify-center" style={{ background: "#0a0a0a" }}>
        <p className="text-gray-500 text-sm">Select a package node to view AI analysis</p>
      </div>
    );
  }

  if (!assessment) {
    return (
      <div className="h-full flex items-center justify-center" style={{ background: "#0a0a0a" }}>
        <div className="text-center space-y-2">
          <Shield className="w-8 h-8 text-green-500 mx-auto opacity-60" />
          <p className="text-gray-400 text-sm">No anomalies detected</p>
          <p className="text-gray-600 text-xs">Clean behavioral diff — package treated as safe</p>
        </div>
      </div>
    );
  }

  const isMalicious = assessment.is_malicious;
  const confidencePct = Math.round(assessment.confidence * 100);

  const accentColor = isMalicious ? "#ef4444" : "#22c55e";
  const accentBg = isMalicious ? "#ef444415" : "#22c55e15";
  const accentBorder = isMalicious ? "#ef4444" : "#22c55e";

  return (
    <div className="h-full flex flex-col overflow-hidden text-gray-200" style={{ background: "#0a0a0a" }}>
      {/* Header */}
      <div className="p-4 border-b border-[#374151] bg-[#111827]">
        <div className="flex items-center gap-3">
          {isMalicious ? (
            <ShieldAlert className="w-6 h-6 text-red-500 flex-shrink-0" />
          ) : (
            <Shield className="w-6 h-6 text-green-500 flex-shrink-0" />
          )}
          <div className="flex-1 min-w-0">
            <div className="flex items-center justify-between gap-2">
              <span
                className="text-lg font-bold"
                style={{ color: accentColor }}
              >
                {isMalicious ? "MALICIOUS" : "SAFE"}
              </span>
              <span
                className="text-xs px-2 py-1 rounded border font-mono"
                style={{
                  color: accentColor,
                  background: accentBg,
                  borderColor: accentBorder,
                }}
              >
                {confidencePct}% confidence
              </span>
            </div>
            <p className="text-xs text-gray-500 mt-0.5">{selectedNode}</p>
          </div>
        </div>
      </div>

      {/* Confidence bar */}
      <div className="px-4 pt-4">
        <div className="flex items-center justify-between text-xs text-gray-500 mb-1">
          <span>Confidence</span>
          <span style={{ color: accentColor }}>{confidencePct}%</span>
        </div>
        <div className="w-full h-1.5 bg-gray-800 rounded-full overflow-hidden">
          <div
            className="h-full rounded-full transition-all"
            style={{
              width: `${confidencePct}%`,
              background: accentColor,
            }}
          />
        </div>
      </div>

      {/* Body */}
      <div className="flex-1 overflow-y-auto p-4 space-y-4">
        {/* Justification */}
        <div className="rounded-lg border border-[#374151] bg-[#111827] p-4">
          <div className="text-xs uppercase tracking-wider text-gray-500 mb-2">Assessment</div>
          <p className="text-sm text-gray-200 leading-relaxed">{assessment.justification}</p>
        </div>

        {/* Indicators */}
        {assessment.indicators && assessment.indicators.length > 0 && (
          <div className="rounded-lg border border-[#374151] bg-[#111827] p-4">
            <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-gray-500 mb-3">
              <AlertTriangle className="w-3.5 h-3.5 text-orange-400" />
              <span>Indicators ({assessment.indicators.length})</span>
            </div>
            <ul className="space-y-2">
              {assessment.indicators.map((indicator, idx) => (
                <li key={idx} className="flex items-start gap-2 text-sm">
                  <span
                    className="mt-1.5 w-1.5 h-1.5 rounded-full flex-shrink-0"
                    style={{ background: isMalicious ? "#ef4444" : "#22c55e" }}
                  />
                  <span className="text-gray-300">{indicator}</span>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="p-3 bg-[#111827] border-t border-[#374151] flex justify-between items-center text-[10px] uppercase tracking-wider text-gray-500">
        <span>AI Security Assessment</span>
        {isMalicious ? (
          <ShieldAlert className="w-4 h-4 text-red-500 opacity-70" />
        ) : (
          <Shield className="w-4 h-4 text-green-500 opacity-50" />
        )}
      </div>
    </div>
  );
}
