import React, { useState, useMemo } from 'react';
import { TableInput, TableOutput } from '@/types/schemas';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { ScrollArea } from '@/components/ui/scroll-area';
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from '@/components/ui/table';
import { Loader2, Search, ArrowUpDown, CheckSquare, Square } from 'lucide-react';
import { cn } from '@/lib/utils';

interface Props {
  requestId: string;
  input: TableInput;
  onSubmit: (output: TableOutput) => Promise<void>;
  loading?: boolean;
}

export const TableDialog: React.FC<Props> = ({ input, onSubmit, loading }) => {
  const [selectedIds, setSelectedIds] = useState<Set<number | string>>(new Set());
  const [search, setSearch] = useState('');
  const [sortConfig, setSortConfig] = useState<{ key: string; direction: 'asc' | 'desc' } | null>(null);
  const [submitting, setSubmitting] = useState(false);

  // Derive columns from data if not provided
  const columns = useMemo(() => {
    if (input.columns && input.columns.length > 0) return input.columns;
    if (input.data && input.data.length > 0) return Object.keys(input.data[0]);
    return [];
  }, [input.columns, input.data]);

  // Filter and sort data
  const processedData = useMemo(() => {
    let data = [...input.data];

    if (search && input.searchable) {
      const lowerSearch = search.toLowerCase();
      data = data.filter(row => 
        Object.values(row).some(val => 
          String(val).toLowerCase().includes(lowerSearch)
        )
      );
    }

    if (sortConfig) {
      data.sort((a, b) => {
        const aVal = a[sortConfig.key];
        const bVal = b[sortConfig.key];
        
        if (aVal < bVal) return sortConfig.direction === 'asc' ? -1 : 1;
        if (aVal > bVal) return sortConfig.direction === 'asc' ? 1 : -1;
        return 0;
      });
    }

    return data;
  }, [input.data, search, sortConfig, input.searchable]);

  const handleSort = (key: string) => {
    setSortConfig(current => {
      if (current?.key === key) {
        return { key, direction: current.direction === 'asc' ? 'desc' : 'asc' };
      }
      return { key, direction: 'asc' };
    });
  };

  const toggleSelection = (row: any) => {
    const id = row.id || JSON.stringify(row);
    const newSelected = new Set(selectedIds);
    
    if (input.multiSelect) {
      if (newSelected.has(id)) {
        newSelected.delete(id);
      } else {
        newSelected.add(id);
      }
    } else {
      newSelected.clear();
      newSelected.add(id);
    }
    
    setSelectedIds(newSelected);
  };

  const toggleAll = () => {
    if (selectedIds.size === processedData.length) {
      setSelectedIds(new Set());
    } else {
      const newSelected = new Set();
      processedData.forEach(row => {
        newSelected.add(row.id || JSON.stringify(row));
      });
      setSelectedIds(newSelected as Set<string | number>);
    }
  };

  const handleSubmit = async () => {
    if (selectedIds.size === 0) return;
    
    setSubmitting(true);
    
    // Find full objects for selected IDs
    const selectedObjects = input.data.filter(row => 
      selectedIds.has(row.id || JSON.stringify(row))
    );

    await onSubmit({
      selected: input.multiSelect ? selectedObjects : selectedObjects[0]
    });
    setSubmitting(false);
  };

  return (
    <div className="bg-background p-6 md:p-8 min-h-[500px] flex flex-col relative">
      <div className="space-y-4 mb-6">
        <h2 className="text-2xl font-display font-bold tracking-tight text-primary uppercase">
          {input.title}
        </h2>
        <div className="h-px w-full bg-border" />
      </div>

      {input.searchable && (
        <div className="relative mb-4 group">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground group-focus-within:text-primary transition-colors" />
          <Input 
            placeholder="SEARCH_DATA..." 
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="pl-9 cyber-input h-12 font-mono text-sm"
          />
        </div>
      )}

      <div className="flex-1 border border-border bg-black/20 mb-6 overflow-hidden flex flex-col">
        <div className="overflow-auto flex-1">
          <Table>
            <TableHeader className="bg-muted/50 sticky top-0 z-10">
              <TableRow className="hover:bg-transparent border-border">
                <TableHead className="w-[50px] text-center">
                  {input.multiSelect && (
                    <div 
                      className="cursor-pointer hover:text-primary transition-colors"
                      onClick={toggleAll}
                    >
                      {selectedIds.size > 0 && selectedIds.size === processedData.length ? (
                        <CheckSquare className="h-4 w-4" />
                      ) : (
                        <Square className="h-4 w-4" />
                      )}
                    </div>
                  )}
                </TableHead>
                {columns.map(col => (
                  <TableHead 
                    key={col} 
                    className="font-mono text-xs uppercase tracking-wider cursor-pointer hover:text-primary transition-colors select-none"
                    onClick={() => handleSort(col)}
                  >
                    <div className="flex items-center gap-2">
                      {col}
                      <ArrowUpDown className="h-3 w-3 opacity-50" />
                    </div>
                  </TableHead>
                ))}
              </TableRow>
            </TableHeader>
            <TableBody>
              {processedData.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={columns.length + 1} className="h-24 text-center text-muted-foreground font-mono">
                    NO_DATA_AVAILABLE
                  </TableCell>
                </TableRow>
              ) : (
                processedData.map((row, idx) => {
                  const id = row.id || JSON.stringify(row);
                  const isSelected = selectedIds.has(id);
                  
                  return (
                    <TableRow 
                      key={idx}
                      className={cn(
                        "cursor-pointer transition-colors border-border/50 hover:bg-primary/5",
                        isSelected && "bg-primary/10"
                      )}
                      onClick={() => toggleSelection(row)}
                    >
                      <TableCell className="text-center">
                        <div className={cn(
                          "transition-colors",
                          isSelected ? "text-primary" : "text-muted-foreground"
                        )}>
                          {isSelected ? (
                            <CheckSquare className="h-4 w-4" />
                          ) : (
                            <Square className="h-4 w-4" />
                          )}
                        </div>
                      </TableCell>
                      {columns.map(col => (
                        <TableCell key={col} className="font-mono text-sm">
                          {String(row[col])}
                        </TableCell>
                      ))}
                    </TableRow>
                  );
                })
              )}
            </TableBody>
          </Table>
        </div>
      </div>

      <div className="flex justify-end pt-4 border-t border-border">
        <div className="flex items-center gap-4 w-full">
          <div className="text-xs font-mono text-muted-foreground flex-1">
            {selectedIds.size} SELECTED / {processedData.length} TOTAL
          </div>
          <Button 
            className="cyber-button min-w-[140px]"
            onClick={handleSubmit}
            disabled={loading || submitting || selectedIds.size === 0}
          >
            {submitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
            SUBMIT_SELECTION
          </Button>
        </div>
      </div>
    </div>
  );
};
