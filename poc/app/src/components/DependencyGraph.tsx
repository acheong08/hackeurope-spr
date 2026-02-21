import { useEffect } from "react";
import {
  ReactFlow,
  Controls,
  Background,
  type Node,
  type Edge,
  type OnNodesChange,
  OnEdgesChange,
  useReactFlow,
} from "@xyflow/react";
import { Progress } from "./ui/progress";

interface DependencyGraphProps {
  progress: number;
  onNodeClick: (nodeId: string) => void;
  nodes: Node[];
  edges: Edge[];
  onNodesChange: OnNodesChange<Node>;
  onEdgesChange: OnEdgesChange<Edge>;
  showDirectOnly: boolean;
  onToggleDirectOnly: () => void;
  directDepCount: number;
  totalDepCount: number;
}

// Component to handle fit view when filter changes - must be inside ReactFlow
function FitViewOnChange({ showDirectOnly }: { showDirectOnly: boolean }) {
  const { fitView } = useReactFlow();
  
  useEffect(() => {
    const timer = setTimeout(() => {
      fitView({ padding: 0.2, duration: 500 });
    }, 50);
    return () => clearTimeout(timer);
  }, [showDirectOnly, fitView]);
  
  return null;
}

export function DependencyGraph({
  progress,
  onNodeClick,
  nodes,
  edges,
  onNodesChange,
  onEdgesChange,
  showDirectOnly,
  onToggleDirectOnly,
  directDepCount,
  totalDepCount,
}: DependencyGraphProps) {
  return (
    <div className="h-full flex flex-col">
      <div
        className="p-4 border-b"
        style={{
          borderColor: "#374151",
          background: "#111827",
        }}
      >
        <div className="flex justify-between items-center mb-3">
          <h2 className="text-lg" style={{ color: "#22c55e" }}>
            Dependency Analysis
          </h2>
          <button
            onClick={onToggleDirectOnly}
            className="px-3 py-1.5 text-xs rounded border transition-colors cursor-pointer"
            style={{
              background: showDirectOnly ? "#22c55e" : "transparent",
              color: showDirectOnly ? "#000" : "#22c55e",
              borderColor: "#22c55e",
            }}
          >
            {showDirectOnly 
              ? `Show All (${totalDepCount})` 
              : `Show Direct Only (${directDepCount})`
            }
          </button>
        </div>
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
          fitViewOptions={{ padding: 0.2, duration: 800 }}
          minZoom={0.1}
          maxZoom={2}
          style={{ background: "#0a0a0a" }}
        >
          <FitViewOnChange showDirectOnly={showDirectOnly} />
          <Background gap={16} />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  );
}
