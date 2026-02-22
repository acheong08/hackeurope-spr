import React, { useState, useMemo, useCallback, useEffect } from 'react';
import { 
  ReactFlow, 
  Background, 
  Controls, 
  type Node, 
  type Edge,
  type OnNodesChange,
  type OnEdgesChange
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { getLayoutedElements } from '../utils/getLayoutedElements';

interface DependencyGraphProps {
  progress: number;
  nodes: Node[];
  edges: Edge[];
  onNodesChange: OnNodesChange;
  onEdgesChange: OnEdgesChange;
  expandedNodeIds: Set<string>;
  setExpandedNodeIds: React.Dispatch<React.SetStateAction<Set<string>>>;
  handleNodeClick: (_: React.MouseEvent, node: Node) => void;
}

export function DependencyGraph({
  progress,
  nodes: allNodes,
  edges: allEdges,
  onNodesChange,
  onEdgesChange,
  expandedNodeIds,
  setExpandedNodeIds,
  handleNodeClick
}: DependencyGraphProps) {
  const rootNodeIds = useMemo(() => {
    const targets = new Set(allEdges.map((e) => e.target));
    const sources = new Set(allEdges.map((e) => e.source));
    return allNodes.filter((n) => !targets.has(n.id) && sources.has(n.id)).map((n) => n.id).slice(0, 10);
  }, [allNodes, allEdges]);

  const { visibleNodes, visibleEdges } = useMemo(() => {
    const visibleIds = new Set<string>();
    const stack = [...rootNodeIds];

    while (stack.length > 0) {
      const currentId = stack.pop()!;

      if (expandedNodeIds.has(currentId)) {
        visibleIds.add(currentId);
        const children = allEdges
          .filter((edge) => edge.source == currentId)
          .map((edge) => edge.target).slice(0, 10);

        children.forEach((childId) => {
          stack.push(childId);
        });
      }
    }

    const filteredNodes = allNodes.filter((n) => visibleIds.has(n.id));
    const filteredEdges = allEdges.filter(
      (e) => visibleIds.has(e.source) && visibleIds.has(e.target)
    );

    return {
      visibleNodes: getLayoutedElements(filteredNodes, filteredEdges),
      visibleEdges: filteredEdges,
    };
  }, [allNodes, allEdges, expandedNodeIds, rootNodeIds]);

  useEffect(() => {
    setExpandedNodeIds((prev) => {
      if (prev.size === 0) return new Set(rootNodeIds);
      return prev;
    });
  }, [rootNodeIds]);

  return (
    <div className="h-full flex flex-col">
      <div className="p-4 border-b border-[#374151] bg-[#111827]">
        <h2 className="text-lg mb-3 text-[#22c55e]">Dependency Analysis</h2>
        <div className="space-y-2">
          <div className="flex justify-between text-sm">
            <span className="text-[#9ca3af]">Progress</span>
            <span className="text-[#22c55e]">{Math.round(progress)}%</span>
          </div>
        </div>
      </div>
      <div className="flex-1 bg-[#0a0a0a]">
        <ReactFlow
          nodes={visibleNodes}
          edges={visibleEdges}
          onNodesChange={onNodesChange}
          onEdgesChange={onEdgesChange}
          onNodeClick={handleNodeClick}
          fitView
          fitViewOptions={{ padding: 0.2, duration: 200 }}
          style={{ background: "#0a0a0a" }}
        >
          <Background gap={16} color="#333" />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  );
}