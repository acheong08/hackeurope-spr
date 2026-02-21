import { useEffect, useContext } from 'react';
import { SocketContext } from '../providers/SocketProvider';
import {
  ReactFlow,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  MarkerType,
  type Node,
  type Edge,
} from '@xyflow/react';
import type { Dependency } from '../types/Dependency';
import { getLayoutedElements } from '../utils/getLayoutedElements';
import { Progress } from './ui/progress';

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

interface DependencyGraph {
  progress: number
  onNodeClick: (nodeId: string) => void;
}

export function DependencyGraph({ progress, onNodeClick  }: DependencyGraph) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);
  const socket = useContext(SocketContext);

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
    socket?.on("dependency-graph", constructDependencyGraph);
    constructDependencyGraph([
      { label: 'kleur@4.1.5' }, 
      { label: 'nanoid@3.3.10', dependent: 'kleur@4.1.5' }, 
      { label: 'test@4.2.10', dependent: 'kleur@4.1.5' }
    ]);

    return () => {
      socket?.off("dependency-graph", constructDependencyGraph);
    }
  }, [socket]);

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b" style={{ 
        borderColor: '#374151',
        background: '#111827'
      }}>
        <h2 className="text-lg mb-3" style={{ color: '#22c55e' }}>
          Dependency Analysis
        </h2>
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span style={{ color: '#9ca3af' }}>Progress</span>
            <span style={{ color: '#22c55e' }}>{Math.round(progress)}%</span>
          </div>
          <Progress 
            value={progress} 
            className="h-2"
            indicatorClassName={'bg-green-500'}
            style={{ background: '#374151' }}
          />
          <div className="flex gap-4 text-xs mt-3 flex-wrap">
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded" style={{ background: '#22c55e' }} />
              <span style={{ color: '#9ca3af' }}>Safe</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded" style={{ background: '#ef4444' }} />
              <span style={{ color: '#9ca3af' }}>Flagged</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded" style={{ background: '#374151' }} />
              <span style={{ color: '#9ca3af' }}>Pending</span>
            </div>
          </div>
        </div>
      </div>
      <div className="flex-1" style={{ background: '#0a0a0a' }}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        onNodeClick={(_, node) => onNodeClick(node.id)}
        fitView
        style={{ background: '#0a0a0a' }}
      >
        <Background gap={16} />
        <Controls />
      </ReactFlow>
      </div>
    </div>
  );
}