import { useState } from "react";
import { DependencyGraph } from "./components/DependencyGraph";
import { Terminal } from "./components/Terminal";
import { Analysis } from "./components/Analysis";
import { Data } from "./components/Data";
import Header from "./components/Header";
import {
  ResizablePanelGroup,
  ResizablePanel,
  ResizableHandle,
} from "./components/ui/resizable";
import {
  useNodesState,
  useEdgesState,
  MarkerType,
  type Node,
  type Edge,
} from "@xyflow/react";
import type { Dependency } from "./types/Dependency";
import { getLayoutedElements } from "./utils/getLayoutedElements";

enum Tab {
  LOGS,
  DATA,
  ANALYSIS
}

const safePkgStyle = {
  background: "#22c55e",
  color: "#000",
  border: "2px solid #4ade80",
  borderRadius: "8px",
  padding: "10px",
};

const flaggedPkgStyle = {
  background: "#ef4444",
  color: "#fff",
  border: "2px solid #dc2626",
  borderRadius: "8px",
  padding: "10px",
};

const dataGatheringPkgStyle = {
  background: "#1f2937",
  color: "#9ca3af",
  border: "2px solid #374151",
  borderRadius: "8px",
  padding: "10px",
};

export default function App() {
  const [progress, setProgress] = useState(0);
  const [logs, setLogs] = useState<string[]>([]);
  const [analysisKey, setAnalysisKey] = useState(0);

  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [uploadedFile, setUploadedFile] = useState<File | null>(null);
  const [packageData, setPackageData] = useState<any>(null);

  const [selectedTab, setSelectedTab] = useState<Tab>(Tab.LOGS);

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const startAnalysis = () => {
    setProgress(0);
    setLogs([]);
    setAnalysisKey((prev) => prev + 1);
    setSelectedTab(Tab.LOGS);
  };

  const addLog = (log: string) => {
    setLogs((curLogs) => [...curLogs, log]);
  };

  const handleNodeClick = (nodeId: string) => {
    if (selectedNode == nodeId) {
      setSelectedNode(null);
      setSelectedTab(Tab.LOGS);
    } else {
      setSelectedNode(nodeId);
    }
  };

  const handleSetPanel = (tab: Tab) => {
    setSelectedTab(tab);
  };

  const handleFileUpload = (file: File) => {
    setUploadedFile(file);

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const content = e.target?.result as string;
        const json = JSON.parse(content);
        setPackageData(json);

        setLogs((prev) => [
          ...prev,
          `\n> Uploaded: ${file.name}`,
          `> Ready to analyze package with ${Object.keys(json.dependencies || {}).length + Object.keys(json.devDependencies || {}).length} dependencies`,
        ]);
      } catch (error) {
        alert("Error parsing package.json: " + (error as Error).message);
        setUploadedFile(null);
      }
    };
    reader.readAsText(file);
  };

  const handleFileRemove = () => {
    setUploadedFile(null);
    setPackageData(null);
  };

  const constructDependencyGraph = (dependencies: Dependency[]) => {
    for (const dependency of dependencies) {
      updateGraph(dependency);
    }
  };

  const updateGraph = (dependency: Dependency) => {
    const newNode: Node = {
      id: dependency.label,
      type: "default",
      data: { label: dependency.label },
      position: { x: 0, y: 0 },
      style: dataGatheringPkgStyle,
    };

    let newEdge: Edge | null = null;
    if (dependency.dependent) {
      newEdge = {
        id: `${dependency.dependent}-${dependency.label}`,
        source: dependency.dependent,
        target: dependency.label,
        markerEnd: { type: MarkerType.ArrowClosed },
      };
    }

    setEdges((curEdges: Edge[]) => {
      const nextEdges = newEdge ? [...curEdges, newEdge] : curEdges;

      setNodes((curNodes: Node[]) => {
        const layoutedNodes = getLayoutedElements(
          [...curNodes, newNode],
          nextEdges,
        );
        return layoutedNodes;
      });

      return nextEdges;
    });
  };

  return (
    <div
      className="h-screen flex flex-col"
      style={{
        background: "#0a0a0a",
        color: "#22c55e",
      }}
    >
      <Header
        startAnalysis={startAnalysis}
        uploadedFile={uploadedFile}
        onFileUpload={handleFileUpload}
        onFileRemove={handleFileRemove}
      />
      <div className="flex-1 overflow-hidden">
        <ResizablePanelGroup direction="horizontal" className="h-full">
          <ResizablePanel defaultSize={65} minSize={30}>
            <div className="h-full border-r" style={{ borderColor: "#374151" }}>
              <DependencyGraph
                progress={progress}
                onNodeClick={handleNodeClick}
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
              />
            </div>
          </ResizablePanel>

          <ResizableHandle
            style={{
              width: "2px",
              background: "#374151",
            }}
          />
          <ResizablePanel defaultSize={35} minSize={30}>
            <div className="h-full flex flex-col">
              <div
                className="flex border-b"
                style={{
                  borderColor: "#374151",
                  background: "#111827",
                }}
              >
                <button
                  onClick={() => handleSetPanel(Tab.LOGS)}
                  className="flex-1 px-4 py-3 text-sm transition-colors cursor-pointer"
                  style={{
                    background:
                      selectedTab == Tab.LOGS
                        ? "#0a0a0a"
                        : "transparent",
                    color:
                      selectedTab == Tab.LOGS ? "#22c55e" : "#9ca3af",
                    borderBottom:
                      selectedTab == Tab.LOGS
                        ? `2px solid #22c55e`
                        : "none",
                  }}
                >
                  Logs
                </button>
                {selectedNode && (
                  <>
                    <button
                      className="flex-1 px-4 py-3 text-sm transition-colors cursor-pointer"
                      onClick={() => handleSetPanel(Tab.DATA)}
                      style={{
                        background:
                          selectedTab == Tab.DATA
                            ? "#0a0a0a"
                            : "transparent",
                        color:
                          selectedTab == Tab.DATA
                            ? "#22c55e"
                            : "#9ca3af",
                        borderBottom:
                          selectedTab == Tab.DATA
                            ? `2px solid #22c55e`
                            : "none",
                      }}
                    >
                      Data
                    </button>
                    <button
                      className="flex-1 px-4 py-3 text-sm transition-colors cursor-pointer"
                      onClick={() => handleSetPanel(Tab.ANALYSIS)}
                      style={{
                        background:
                          selectedTab == Tab.ANALYSIS
                            ? "#0a0a0a"
                            : "transparent",
                        color:
                          selectedTab == Tab.ANALYSIS
                            ? "#22c55e"
                            : "#9ca3af",
                        borderBottom:
                          selectedTab == Tab.ANALYSIS
                            ? `2px solid #22c55e`
                            : "none",
                      }}
                    >
                      Analysis
                    </button>
                  </>
                )}
              </div>
              {selectedTab == Tab.LOGS ? (
                <Terminal logs={logs} addLog={addLog} />
              ) : selectedTab == Tab.ANALYSIS ? (
                <Analysis selectedNode={selectedNode} />
              ) : 
                <Data selectedNode={selectedNode} />}
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
    </div>
  );
}
