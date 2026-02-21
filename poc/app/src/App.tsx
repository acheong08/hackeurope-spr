import { useState, useEffect, useContext } from "react";
import { DependencyGraph } from "./components/DependencyGraph";
import { Terminal } from "./components/Terminal";
import { Analysis } from "./components/Analysis";
import { Data } from "./components/Data";
import Header from "./components/Header";
import { SocketContext } from "./providers/SocketProvider";
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

interface PackageNode {
  id: string;
  name: string;
  version: string;
  resolved: string;
  integrity: string;
  dependencies: Record<string, string>;
}

export default function App() {
  const [progress, setProgress] = useState(0);
  const [logs, setLogs] = useState<string[]>([]);
  const [analysisKey, setAnalysisKey] = useState(0);
  const [isAnalyzing, setIsAnalyzing] = useState(false);

  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [uploadedFile, setUploadedFile] = useState<File | null>(null);
  const [packageData, setPackageData] = useState<any>(null);
  const [packageContent, setPackageContent] = useState<string>("");

  const [selectedTab, setSelectedTab] = useState<Tab>(Tab.LOGS);

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const { send, subscribe, isConnected } = useContext(SocketContext);

  // Subscribe to WebSocket messages
  useEffect(() => {
    const handleMessage = (msg: { type: string; payload: any }) => {
      switch (msg.type) {
        case "dag": {
          // Received DAG data - build the graph
          const payload = msg.payload as {
            root_package: { id: string; name: string; version: string };
            nodes: PackageNode[];
            edge_count: number;
          };
          
          // Build nodes from DAG
          const newNodes: Node[] = payload.nodes.map((pkg) => ({
            id: pkg.id,
            type: "default",
            data: { label: `${pkg.name}@${pkg.version}` },
            position: { x: 0, y: 0 },
            style: dataGatheringPkgStyle,
          }));

          // Build edges from dependencies
          const newEdges: Edge[] = [];
          
          payload.nodes.forEach((pkg) => {
            // Check if this package is a direct dependency of root
            if (pkg.dependencies && Object.keys(pkg.dependencies).length > 0) {
              // This is likely the root or has deps
              Object.entries(pkg.dependencies).forEach(([depName]) => {
                // Find the dependent node
                const dependentNode = payload.nodes.find(n => n.name === depName);
                if (dependentNode) {
                  newEdges.push({
                    id: `${pkg.id}-${dependentNode.id}`,
                    source: pkg.id,
                    target: dependentNode.id,
                    markerEnd: { type: MarkerType.ArrowClosed },
                  });
                }
              });
            }
          });

          // Layout the graph
          const layoutedNodes = getLayoutedElements(newNodes, newEdges);
          setNodes(layoutedNodes);
          setEdges(newEdges);
          
          addLog(`✓ DAG received: ${payload.nodes.length} packages, ${newEdges.length} dependencies`);
          break;
        }
        
        case "progress": {
          const payload = msg.payload as { percent: number; stage: string; message: string };
          setProgress(payload.percent);
          break;
        }
        
        case "package_status": {
          const payload = msg.payload as { package_id: string; name: string; version: string; status: string };
          // Update node styling based on status
          setNodes((currentNodes) =>
            currentNodes.map((node) => {
              if (node.id === payload.package_id) {
                const style =
                  payload.status === "complete"
                    ? safePkgStyle
                    : payload.status === "failed"
                    ? flaggedPkgStyle
                    : dataGatheringPkgStyle;
                return { ...node, style };
              }
              return node;
            })
          );
          // Log status change
          const statusIcon = payload.status === "complete" ? "✓" : 
                            payload.status === "failed" ? "✗" : "→";
          addLog(`${statusIcon} ${payload.name}@${payload.version}: ${payload.status}`);
          break;
        }
        
        case "complete": {
          const payload = msg.payload as { success: boolean; message: string };
          if (payload.success) {
            addLog(`✓ ${payload.message}`);
          } else {
            addLog(`✗ ${payload.message}`);
          }
          setIsAnalyzing(false);
          break;
        }
        
        case "error": {
          const payload = msg.payload as { message: string };
          addLog(`✗ Error: ${payload.message}`);
          setIsAnalyzing(false);
          break;
        }
        
        case "log": {
          const payload = msg.payload as { message: string; level?: string };
          const prefix = payload.level === "success" ? "✓ " : 
                         payload.level === "warning" ? "⚠ " : 
                         payload.level === "error" ? "✗ " : 
                         payload.level === "info" ? "→ " : "";
          addLog(`${prefix}${payload.message}`);
          break;
        }
      }
    };

    const unsubscribe = subscribe(handleMessage);
    return unsubscribe;
  }, [subscribe]);

  const startAnalysis = () => {
    if (isAnalyzing) {
      addLog("⚠ Analysis already in progress");
      return;
    }
    
    if (!packageContent) {
      addLog("⚠ Please upload a package.json file first");
      return;
    }
    
    if (!isConnected) {
      addLog("⚠ WebSocket not connected");
      return;
    }

    setIsAnalyzing(true);
    setProgress(0);
    setLogs([]);
    setAnalysisKey((prev) => prev + 1);
    setSelectedTab(Tab.LOGS);
    setNodes([]);
    setEdges([]);

    // Send analyze request
    addLog("→ Starting analysis...");
    send({
      type: "analyze",
      payload: {
        package_json: packageContent,
      },
    });
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
        setPackageContent(content);

        setLogs((prev) => [
          ...prev,
          `> Uploaded: ${file.name}`,
          `> Package: ${json.name}@${json.version}`,
          `> Dependencies: ${Object.keys(json.dependencies || {}).length + Object.keys(json.devDependencies || {}).length}`,
          `> Ready to analyze - click "Start Analysis"`,
        ]);
      } catch (error) {
        alert("Error parsing package.json: " + (error as Error).message);
        setUploadedFile(null);
        setPackageContent("");
      }
    };
    reader.readAsText(file);
  };

  const handleFileRemove = () => {
    setUploadedFile(null);
    setPackageData(null);
    setPackageContent("");
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
        isConnected={isConnected}
        isAnalyzing={isAnalyzing}
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
                <Terminal logs={logs} />
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
