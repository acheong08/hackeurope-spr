import { useState, useEffect } from 'react';
import { DependencyGraph } from './components/DependencyGraph';
import { Terminal } from './components/Terminal';
import { VulnerabilityPanel } from './components/VulnerabilityPanel';
import Header from './components/Header';
import { FileUpload } from './components/FileUpload';
import { 
  ResizablePanelGroup, 
  ResizablePanel, 
  ResizableHandle 
} from './components/ui/resizable';
import {
  useNodesState,
  useEdgesState,
  MarkerType,
  type Node,
  type Edge,
} from '@xyflow/react';
import type { Dependency } from './types/Dependency';
import { getLayoutedElements } from './utils/getLayoutedElements';

enum Tab {
  SPR_ANALYSIS,
  VULNERABILITY_DETAILS
}

const safePkgStyle = {
  background: '#22c55e',
  color: '#000',
  border: '2px solid #4ade80',
  borderRadius: '8px',
  padding: '10px',
};

const flaggedPkgStyle = {
  background: '#ef4444',
  color: '#fff',
  border: '2px solid #dc2626',
  borderRadius: '8px',
  padding: '10px',
};

const dataGatheringPkgStyle = {
  background: '#1f2937',
  color: '#9ca3af',
  border: '2px solid #374151',
  borderRadius: '8px',
  padding: '10px',
};

export default function App() {
  const [progress, setProgress] = useState(0);
  const [logs, setLogs] = useState<string[]>([]);
  const [analysisKey, setAnalysisKey] = useState(0);

  const [selectedNode, setSelectedNode] = useState<string | null>(null);
  const [uploadedFile, setUploadedFile] = useState<File | null>(null);
  const [packageData, setPackageData] = useState<any>(null);

  const [selectedTab, setSelectedTab] = useState<Tab>(Tab.SPR_ANALYSIS);

  const [nodes, setNodes, onNodesChange] = useNodesState<Node>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);

  const startAnalysis = () => {
    setProgress(0);
    setLogs([]);
    setAnalysisKey(prev => prev + 1);
    setSelectedTab(Tab.SPR_ANALYSIS);
  };

  const addLog = (log: string) => {
    setLogs((curLogs) => [...curLogs, log]);
  }

  const handleNodeClick = (nodeId: string) => {
    if (selectedNode == nodeId) {
      setSelectedNode(null);
      setSelectedTab(Tab.SPR_ANALYSIS);
    } else {
      setSelectedNode(nodeId);
      setSelectedTab(Tab.VULNERABILITY_DETAILS);
    }
  };

  const handleSetPanel = (tab: Tab) => {
    setSelectedTab(tab);
  };

    const handleFileUpload = (file: File) => {
    setUploadedFile(file);
    
    // Read and parse the file
    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const content = e.target?.result as string;
        const json = JSON.parse(content);
        setPackageData(json);
        
        // Add log that file was uploaded
        setLogs(prev => [
          ...prev,
          `\n> Uploaded: ${file.name}`,
          `> Ready to analyze package with ${Object.keys(json.dependencies || {}).length + Object.keys(json.devDependencies || {}).length} dependencies`,
        ]);
      } catch (error) {
        alert('Error parsing package.json: ' + (error as Error).message);
        setUploadedFile(null);
      }
    };
    reader.readAsText(file);
  };

  const handleFileRemove = () => {
    setUploadedFile(null);
    setPackageData(null);
    setLogs(prev => [...prev, '\n> Package file removed']);
  };

   const constructDependencyGraph = (dependencies: Dependency[]) => {
    for (const dependency of dependencies) {
      updateGraph(dependency);
    }
  }

  const updateGraph = (dependency: Dependency) => {
    const newNode: Node = {
      id: dependency.label,
      type: 'default',
      data: { label: dependency.label },
      position: { x: 0, y: 0 },
      style: dataGatheringPkgStyle
    };
    
    let newEdge: Edge | null = null;
    if (dependency.dependent) {
      newEdge = {
        id: `${dependency.dependent}-${dependency.label}`,
        source: dependency.dependent,
        target: dependency.label,
        markerEnd: { type: MarkerType.ArrowClosed }
      };
    }

    setEdges((curEdges: Edge[]) => {
      const nextEdges = newEdge ? [...curEdges, newEdge] : curEdges;

      setNodes((curNodes: Node[]) => {
        const layoutedNodes = getLayoutedElements([...curNodes, newNode], nextEdges);
        return layoutedNodes;
      });

      return nextEdges;
    });
  };

  useEffect(() => {
    constructDependencyGraph([
      { label: 'kleur@4.1.5' }, 
      { label: 'nanoid@3.3.10', dependent: 'kleur@4.1.5' }, 
      { label: 'test@4.2.10', dependent: 'kleur@4.1.5' }
    ]);
  }, []);

  return (
    <div 
      className="h-screen flex flex-col"
      style={{ 
        background: '#0a0a0a',
        color: '#22c55e'
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
            <div 
              className="h-full border-r"
              style={{ borderColor: '#374151' }}
            >
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
              width: '2px',
              background: '#374151',
            }} 
          />
          <ResizablePanel defaultSize={35} minSize={30}>
            <div className="h-full flex flex-col">
              <div 
                className="flex border-b"
                style={{ 
                  borderColor: '#374151',
                  background: '#111827'
                }}
              >
                
                <button
                  onClick={() => handleSetPanel(Tab.SPR_ANALYSIS)}
                  className="flex-1 px-4 py-3 text-sm transition-colors cursor-pointer"
                  style={{
                    background: selectedTab == Tab.SPR_ANALYSIS ? '#0a0a0a' : 'transparent',
                    color: selectedTab == Tab.SPR_ANALYSIS ? '#22c55e' : '#9ca3af',
                    borderBottom: selectedTab == Tab.SPR_ANALYSIS ? `2px solid #22c55e` : 'none',
                  }}
                >
                  SPR Analysis Terminal
                </button>
                {selectedNode && (
                  <button
                    className="flex-1 px-4 py-3 text-sm transition-colors cursor-pointer"
                    onClick={() => handleSetPanel(Tab.VULNERABILITY_DETAILS)}
                    style={{
                      background: selectedTab == Tab.VULNERABILITY_DETAILS ? '#0a0a0a' : 'transparent',
                      color: selectedTab == Tab.VULNERABILITY_DETAILS ? '#22c55e' : '#9ca3af',
                      borderBottom: selectedTab == Tab.VULNERABILITY_DETAILS ? `2px solid #22c55e` : 'none',
                    }}
                  >
                    Vulnerability Details
                  </button>
                )}
              </div>
              <div className="flex-1 overflow-hidden">
                {selectedTab == Tab.SPR_ANALYSIS ? (
                  <Terminal logs={logs} addLog={addLog} />
                ) : (
                  <VulnerabilityPanel 
                    selectedNode={selectedNode} 
                    onClose={() => handleSetPanel(Tab.SPR_ANALYSIS)} 
                  />
                )}
              </div>
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
    </div>
  );
}
