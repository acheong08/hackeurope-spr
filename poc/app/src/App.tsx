import { useState, useEffect, useContext, useCallback } from "react";
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
  
  // Store full graph data for filtering
  const [allNodes, setAllNodes] = useState<Node[]>([]);
  const [allEdges, setAllEdges] = useState<Edge[]>([]);
  const [directDependencyIds, setDirectDependencyIds] = useState<Set<string>>(new Set());
  const [showDirectOnly, setShowDirectOnly] = useState(false);

  const { send, subscribe, isConnected } = useContext(SocketContext);
  const [expandedNodeIds, setExpandedNodeIds] = useState<Set<string>>(new Set());

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
          
          // Find root node and extract direct dependencies
          const rootNode = payload.nodes.find(n => n.id === payload.root_package.id);
          const directDeps = new Set<string>();
          if (rootNode && rootNode.dependencies) {
            Object.keys(rootNode.dependencies).forEach(depName => {
              const depNode = payload.nodes.find(n => n.name === depName);
              if (depNode) {
                directDeps.add(depNode.id);
              }
            });
          }
          setDirectDependencyIds(directDeps);
          
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
                    animated: true,   // Enable the flow animation
                    markerEnd: { 
                      type: MarkerType.ArrowClosed,
                      color: "#22c55e", // Matches your "Safe" green theme
                    },
                    style: { 
                      stroke: "#22c55e", 
                      strokeWidth: 2 
                    },
                  });
                }
              });
            }
          });

          // Store full graph data
          setAllNodes(newNodes);
          setAllEdges(newEdges);

          // Layout the graph
          const layoutedNodes = getLayoutedElements(newNodes, newEdges);
          setNodes(layoutedNodes);
          setEdges(newEdges);
          setShowDirectOnly(false);
          
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
          // Update style in allNodes
          setAllNodes((currentAllNodes) =>
            currentAllNodes.map((node) => {
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
          // Also update visible nodes
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
    setAllNodes([]);
    setAllEdges([]);
    setDirectDependencyIds(new Set());
    setShowDirectOnly(false);

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

  const handleNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    if (selectedNode == node.id) {
      setSelectedNode(null);
      setSelectedTab(Tab.LOGS);
    } else {
      setSelectedNode(node.id);
    }

    setExpandedNodeIds((prev) => {
      const next = new Set(prev);
      const children = new Set(edges.filter(e => e.source == node.id).map(e => e.target));
      return new Set([...next, ...children]);
    });
  }, [nodes, edges]);

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

  // Toggle between showing all dependencies or only direct ones
  const handleToggleDirectOnly = () => {
    const newShowDirectOnly = !showDirectOnly;
    setShowDirectOnly(newShowDirectOnly);
    
    if (newShowDirectOnly) {
      // Filter to show only root and direct dependencies
      const rootId = allNodes.find(n => n.id.includes("root@"))?.id;
      const visibleIds = new Set([rootId, ...directDependencyIds].filter(Boolean) as string[]);
      
      const filteredNodes = allNodes.filter(n => visibleIds.has(n.id));
      const filteredEdges = allEdges.filter(e => visibleIds.has(e.source) && visibleIds.has(e.target));
      
      // Re-layout the filtered graph
      const layoutedNodes = getLayoutedElements(filteredNodes, filteredEdges);
      setNodes(layoutedNodes);
      setEdges(filteredEdges);
      
      addLog(`→ Showing ${filteredNodes.length} direct dependencies`);
    } else {
      // Show all dependencies
      const layoutedNodes = getLayoutedElements(allNodes, allEdges);
      setNodes(layoutedNodes);
      setEdges(allEdges);
      
      addLog(`→ Showing all ${allNodes.length} dependencies`);
    }
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
                nodes={nodes}
                edges={edges}
                onNodesChange={onNodesChange}
                onEdgesChange={onEdgesChange}
                expandedNodeIds={expandedNodeIds}
                setExpandedNodeIds={setExpandedNodeIds}
                handleNodeClick={handleNodeClick}
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
