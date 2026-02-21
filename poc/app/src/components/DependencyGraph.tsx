import { useEffect } from 'react';
import {
  ReactFlow,
  Controls,
  Background,
  useNodesState,
  useEdgesState,
  type Node,
  type Edge,
} from '@xyflow/react';
import { Progress } from './ui/progress';

interface DependencyGraphProps {
  progress: number;
}

export function DependencyGraph({ progress }: DependencyGraphProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  useEffect(() => {
    const allNodes: Node[] = [
      {
        id: 'root',
        type: 'default',
        position: { x: 250, y: 0 },
        data: { label: 'my-app' },
        style: {
          background: '#22c55e',
          color: '#000',
          border: '2px solid #4ade80',
          borderRadius: '8px',
          padding: '10px',
        },
      },
      {
        id: 'kleur',
        type: 'default',
        position: { x: 50, y: 120 },
        data: { label: 'kleur@4.1.5' },
        style: {
          background: progress > 20 ? '#22c55e' : '#1f2937',
          color: progress > 20 ? '#000' : '#9ca3af',
          border: `2px solid ${progress > 20 ? '#4ade80' : '#374151'}`,
          borderRadius: '8px',
          padding: '10px',
        },
      },
      {
        id: 'nanoid-3.3.10',
        type: 'default',
        position: { x: 200, y: 120 },
        data: { label: 'nanoid@3.3.10' },
        style: {
          background: progress > 40 ? '#22c55e' : '#1f2937',
          color: progress > 40 ? '#000' : '#9ca3af',
          border: `2px solid ${progress > 40 ? '#4ade80' : '#374151'}`,
          borderRadius: '8px',
          padding: '10px',
        },
      },
      {
        id: 'nanoid-3.3.11',
        type: 'default',
        position: { x: 350, y: 120 },
        data: { label: 'nanoid@3.3.11' },
        style: {
          background: progress > 60 ? '#ef4444' : '#1f2937',
          color: progress > 60 ? '#fff' : '#9ca3af',
          border: `2px solid ${progress > 60 ? '#dc2626' : '#374151'}`,
          borderRadius: '8px',
          padding: '10px',
        },
      },
      {
        id: 'example-dep',
        type: 'default',
        position: { x: 125, y: 240 },
        data: { label: 'example.com' },
        style: {
          background: progress > 70 ? '#22c55e' : '#1f2937',
          color: progress > 70 ? '#000' : '#9ca3af',
          border: `2px solid ${progress > 70 ? '#4ade80' : '#374151'}`,
          borderRadius: '8px',
          padding: '10px',
          fontSize: '12px',
        },
      },
      {
        id: 'npmjs-dep',
        type: 'default',
        position: { x: 275, y: 240 },
        data: { label: 'npmjs.com' },
        style: {
          background: progress > 80 ? '#22c55e' : '#1f2937',
          color: progress > 80 ? '#000' : '#9ca3af',
          border: `2px solid ${progress > 80 ? '#4ade80' : '#374151'}`,
          borderRadius: '8px',
          padding: '10px',
          fontSize: '12px',
        },
      },
      {
        id: 'suspicious-dep',
        type: 'default',
        position: { x: 425, y: 240 },
        data: { label: 'iamavirus.com' },
        style: {
          background: progress > 90 ? '#ef4444' : '#1f2937',
          color: progress > 90 ? '#fff' : '#9ca3af',
          border: `2px solid ${progress > 90 ? '#dc2626' : '#374151'}`,
          borderRadius: '8px',
          padding: '10px',
          fontSize: '12px',
        },
      },
    ];

    const allEdges: Edge[] = [
      { id: 'e-root-kleur', source: 'root', target: 'kleur', animated: progress > 10 && progress < 30 },
      { id: 'e-root-nanoid-3.3.10', source: 'root', target: 'nanoid-3.3.10', animated: progress > 30 && progress < 50 },
      { id: 'e-root-nanoid-3.3.11', source: 'root', target: 'nanoid-3.3.11', animated: progress > 50 && progress < 70 },
      { id: 'e-nanoid-3.3.10-example', source: 'nanoid-3.3.10', target: 'example-dep', animated: progress > 60 && progress < 80 },
      { id: 'e-nanoid-3.3.10-npmjs', source: 'nanoid-3.3.10', target: 'npmjs-dep', animated: progress > 70 && progress < 90 },
      { id: 'e-nanoid-3.3.11-suspicious', source: 'nanoid-3.3.11', target: 'suspicious-dep', animated: progress > 80, style: { stroke: progress > 90 ? '#ef4444' : undefined } },
    ];

    setNodes(allNodes);
    setEdges(allEdges);
  }, [progress, setNodes, setEdges]);

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
          fitView
          style={{ background: '#0a0a0a' }}
        >
          <Background color={'#22c55e'} gap={16} />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  );
}