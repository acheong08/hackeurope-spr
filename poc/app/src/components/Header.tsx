import { RotateCcw } from 'lucide-react';

interface HeaderProps {
  startAnalysis: () => void
}

function Header({ startAnalysis }: HeaderProps) {
    return (
      <header 
        className="border-b px-4 md:px-6 py-4 flex flex-col md:flex-row justify-between items-start md:items-center gap-3"
        style={{ 
          borderColor: '#374151',
          background: '#111827'
        }}
      >
        <div className="flex items-center gap-3 flex-wrap">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 rounded flex items-center justify-center" style={{ background: '#22c55e' }}>
              <span className="text-lg" style={{ color: '#000' }}>S</span>
            </div>
            <h1 className="text-xl md:text-2xl" style={{ color: '#22c55e' }}>
              HackEurope SPR
            </h1>
          </div>
          <span className="text-xs md:text-sm px-2 py-1 rounded" style={{ 
            background: '#374151',
            color: '#9ca3af'
          }}>
            Supply Chain Package Registry
          </span>
        </div>
        <div className="flex gap-2 self-end md:self-auto">
          <button
            onClick={startAnalysis}
            className="p-2 rounded-lg transition-colors hover:bg-opacity-80"
            style={{ 
              background: '#374151',
              color: '#22c55e'
            }}
            aria-label="Restart Analysis"
          >
            <RotateCcw className="w-5 h-5" />
          </button>
        </div>
      </header>
  );
}

export default Header;