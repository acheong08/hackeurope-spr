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
  isDark: boolean;
  progress: number;
}

export function DependencyGraph({ isDark, progress }: DependencyGraphProps) {
  const [nodes, setNodes, onNodesChange] = useNodesState([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState([]);

  useEffect(() => {
    // Build the dependency graph based on progress
    const allNodes: Node[] = [
      {
        id: 'root',
        type: 'default',
        position: { x: 250, y: 0 },
        data: { label: 'my-app' },
        style: {
          background: isDark ? '#22c55e' : '#16a34a',
          color: isDark ? '#000' : '#fff',
          border: `2px solid ${isDark ? '#4ade80' : '#22c55e'}`,
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
          background: progress > 20 ? (isDark ? '#22c55e' : '#16a34a') : isDark ? '#1f2937' : '#e5e7eb',
          color: progress > 20 ? (isDark ? '#000' : '#fff') : isDark ? '#9ca3af' : '#6b7280',
          border: `2px solid ${progress > 20 ? (isDark ? '#4ade80' : '#22c55e') : isDark ? '#374151' : '#d1d5db'}`,
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
          background: progress > 40 ? (isDark ? '#22c55e' : '#16a34a') : isDark ? '#1f2937' : '#e5e7eb',
          color: progress > 40 ? (isDark ? '#000' : '#fff') : isDark ? '#9ca3af' : '#6b7280',
          border: `2px solid ${progress > 40 ? (isDark ? '#4ade80' : '#22c55e') : isDark ? '#374151' : '#d1d5db'}`,
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
          background: progress > 60 ? '#ef4444' : isDark ? '#1f2937' : '#e5e7eb',
          color: progress > 60 ? '#fff' : isDark ? '#9ca3af' : '#6b7280',
          border: `2px solid ${progress > 60 ? '#dc2626' : isDark ? '#374151' : '#d1d5db'}`,
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
          background: progress > 70 ? (isDark ? '#22c55e' : '#16a34a') : isDark ? '#1f2937' : '#e5e7eb',
          color: progress > 70 ? (isDark ? '#000' : '#fff') : isDark ? '#9ca3af' : '#6b7280',
          border: `2px solid ${progress > 70 ? (isDark ? '#4ade80' : '#22c55e') : isDark ? '#374151' : '#d1d5db'}`,
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
          background: progress > 80 ? (isDark ? '#22c55e' : '#16a34a') : isDark ? '#1f2937' : '#e5e7eb',
          color: progress > 80 ? (isDark ? '#000' : '#fff') : isDark ? '#9ca3af' : '#6b7280',
          border: `2px solid ${progress > 80 ? (isDark ? '#4ade80' : '#22c55e') : isDark ? '#374151' : '#d1d5db'}`,
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
          background: progress > 90 ? '#ef4444' : isDark ? '#1f2937' : '#e5e7eb',
          color: progress > 90 ? '#fff' : isDark ? '#9ca3af' : '#6b7280',
          border: `2px solid ${progress > 90 ? '#dc2626' : isDark ? '#374151' : '#d1d5db'}`,
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
  }, [isDark, progress, setNodes, setEdges]);

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b" style={{ 
        borderColor: isDark ? '#374151' : '#e5e7eb',
        background: isDark ? '#111827' : '#f9fafb'
      }}>
        <h2 className="text-lg mb-3" style={{ color: isDark ? '#22c55e' : '#16a34a' }}>
          Dependency Analysis
        </h2>
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span style={{ color: isDark ? '#9ca3af' : '#6b7280' }}>Progress</span>
            <span style={{ color: isDark ? '#22c55e' : '#16a34a' }}>{Math.round(progress)}%</span>
          </div>
          <Progress 
            value={progress} 
            className="h-2"
            indicatorClassName={isDark ? 'bg-green-500' : 'bg-green-600'}
            style={{
              background: isDark ? '#374151' : '#e5e7eb',
            }}
          />
          <div className="flex gap-4 text-xs mt-3 flex-wrap">
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded" style={{ background: isDark ? '#22c55e' : '#16a34a' }} />
              <span style={{ color: isDark ? '#9ca3af' : '#6b7280' }}>Safe</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded" style={{ background: '#ef4444' }} />
              <span style={{ color: isDark ? '#9ca3af' : '#6b7280' }}>Flagged</span>
            </div>
            <div className="flex items-center gap-1.5">
              <div className="w-3 h-3 rounded" style={{ background: isDark ? '#374151' : '#e5e7eb' }} />
              <span style={{ color: isDark ? '#9ca3af' : '#6b7280' }}>Pending</span>
            </div>
          </div>
        </div>
      </div>
      <div className="flex-1" style={{ background: isDark ? '#0a0a0a' : '#ffffff' }}>
        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          fitView
          style={{
            background: isDark ? '#0a0a0a' : '#ffffff',
          }}
        >
          <Background color={isDark ? '#22c55e' : '#86efac'} gap={16} />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  );
}