import {
  ReactFlow,
  Controls,
  Background,
  type Node,
  type Edge,
  type OnNodesChange,
  OnEdgesChange,
} from "@xyflow/react";
import { Progress } from "./ui/progress";

interface DependencyGraph {
  progress: number;
  onNodeClick: (nodeId: string) => void;
  nodes: Node[];
  edges: Edge[];
  onNodesChange: OnNodesChange<Node>;
  onEdgesChange: OnEdgesChange<Edge>;
}

export function DependencyGraph({
  progress,
  onNodeClick,
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
}: DependencyGraph) {
  return (
    <div className="h-full flex flex-col">
      <div
        className="p-4 border-b"
        style={{
          borderColor: "#374151",
          background: "#111827",
        }}
      >
        <h2 className="text-lg mb-3" style={{ color: "#22c55e" }}>
          Dependency Analysis
        </h2>
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span style={{ color: "#9ca3af" }}>Progress</span>
            <span style={{ color: "#22c55e" }}>{Math.round(progress)}%</span>
          </div>
          <Progress
            value={progress}
            className="h-2"
            indicatorClassName={"bg-green-500"}
            style={{ background: "#374151" }}
          />
          <div className="flex gap-4 text-xs mt-3 flex-wrap">
            <div className="flex items-center gap-1.5">
              <div
                className="w-3 h-3 rounded"
                style={{ background: "#22c55e" }}
              />
              <span style={{ color: "#9ca3af" }}>Safe</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div
                className="w-3 h-3 rounded"
                style={{ background: "#ef4444" }}
              />
              <span style={{ color: "#9ca3af" }}>Flagged</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div
                className="w-3 h-3 rounded"
                style={{ background: "#374151" }}
              />
              <span style={{ color: "#9ca3af" }}>Pending</span>
            </div>
          </div>
        </div>
      </div>
      <div className="flex-1" style={{ background: "#0a0a0a" }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onNodeClick={(_, node: Node) => onNodeClick(node.id)}
          fitView
          style={{ background: "#0a0a0a" }}
        >
          <Background gap={16} />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  );
}
