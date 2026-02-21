import { useState, useEffect } from 'react';
import { DependencyGraph } from './components/DependencyGraph';
import { Terminal } from './components/Terminal';
import Header from './components/Header';
import SocketProvider from './providers/SocketProvider';

export default function App() {
  const [progress, setProgress] = useState(0);
  const [logs, setLogs] = useState<string[]>([]);
  const [analysisKey, setAnalysisKey] = useState(0);

  const restartAnalysis = () => {
    setProgress(0);
    setLogs([]);
    setAnalysisKey(prev => prev + 1);
  };

  const analysisSteps = [
    { time: 0, progress: 0, log: '$ spr check ./package.json' },
    { time: 500, progress: 5, log: '> SPR - Supply Chain Package Registry v1.0.0' },
    { time: 1000, progress: 10, log: '> Parsing package.json...' },
    { time: 1500, progress: 15, log: '> Found 4 dependencies' },
    { time: 2000, progress: 20, log: '> Building dependency DAG...' },
    { time: 2500, progress: 25, log: '\n→ Analyzing: kleur@4.1.5' },
    { time: 3000, progress: 30, log: '  Mirroring to unsafe registry...' },
    { time: 3500, progress: 35, log: '  Starting behavioral analysis (Tracee)...' },
    { time: 4000, progress: 40, log: '  Files read: /etc/os-release (NORMAL)' },
    { time: 4500, progress: 45, log: '  Network: npmjs.com (SEEN BEFORE)' },
    { time: 5000, progress: 50, log: '✓ kleur@4.1.5 SAFE - Promoted to safe registry\n' },
    { time: 5500, progress: 55, log: '→ Analyzing: nanoid@3.3.10' },
    { time: 6000, progress: 60, log: '  Mirroring to unsafe registry...' },
    { time: 6500, progress: 65, log: '  Starting behavioral analysis (Tracee)...' },
    { time: 7000, progress: 70, log: '  Files read: /etc/os-release (NORMAL)' },
    { time: 7500, progress: 75, log: '  Network: npmjs.com, example.com (SEEN BEFORE)' },
    { time: 8000, progress: 80, log: '✓ nanoid@3.3.10 SAFE - Promoted to safe registry\n' },
    { time: 8500, progress: 85, log: '→ Analyzing: nanoid@3.3.11' },
    { time: 9000, progress: 87, log: '  Mirroring to unsafe registry...' },
    { time: 9500, progress: 89, log: '  Starting behavioral analysis (Tracee)...' },
    { time: 10000, progress: 91, log: '  Files read:' },
    { time: 10200, progress: 92, log: '    SUSPICIOUS: /etc/passwd' },
    { time: 10400, progress: 93, log: '    SUSPICIOUS: ~/.ssh/*' },
    { time: 10800, progress: 94, log: '  Network traffic:' },
    { time: 11000, progress: 95, log: '    - git.github.com (SEEN BEFORE)' },
    { time: 11200, progress: 96, log: '    - randomguy.github.io (SEEN BEFORE)' },
    { time: 11600, progress: 97, log: '    - iamavirus.com ???? SUSPICIOUS - NEW BEHAVIOR' },
    { time: 12000, progress: 98, log: '\n⚠ nanoid@3.3.11 FLAGGED - Suspicious behavior detected' },
    { time: 12500, progress: 99, log: '⚠ Blocking dependent chain' },
    { time: 13000, progress: 100, log: '⚠ Analysis complete with warnings\n' },
    { time: 13500, progress: 100, log: 'Summary:' },
    { time: 14000, progress: 100, log: '  ✓ Safe packages: 2' },
    { time: 14500, progress: 100, log: '  ⚠ Flagged packages: 1' },
    { time: 15000, progress: 100, log: '  → Review flagged packages before deployment' },
  ];

  useEffect(() => {
    const timers: NodeJS.Timeout[] = [];
    
    analysisSteps.forEach((step) => {
      const timer = setTimeout(() => {
        setProgress(step.progress);
        setLogs((prev) => [...prev, step.log]);
      }, step.time);
      timers.push(timer);
    });

    return () => {
      timers.forEach(timer => clearTimeout(timer));
    };
  }, [analysisKey]);

  return (
    <SocketProvider>
      <div 
      className="h-screen flex flex-col"
      style={{ 
        background: '#0a0a0a',
        color: '#22c55e'
      }}
      >
        <Header restartAnalysis={restartAnalysis} />
        <div className="flex-1 flex flex-col lg:grid lg:grid-cols-2 overflow-hidden">
          <div 
            className="h-1/2 lg:h-full border-b lg:border-b-0 lg:border-r"
            style={{ borderColor: '#374151' }}
          >
            <DependencyGraph progress={progress} />
          </div>
          <div className="h-1/2 lg:h-full">
            <Terminal logs={logs} />
          </div>
        </div>
      </div>
    </SocketProvider>
  );
}