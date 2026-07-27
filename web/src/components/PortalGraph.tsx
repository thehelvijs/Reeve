import { useMemo } from 'react';
import { ReactFlow, Background, Controls, Handle, Position, type Node, type Edge } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { Host, Tool, ToolStatus } from '../api';
import { groupToolsByHost } from '../lib/group';
import EntityIcon from './EntityIcon';
import StatusPill from './StatusPill';
import { dotClass, hostTone, TOOL_TONE } from '../lib/statusTone';

// A node carries either a host status or a tool status, so it resolves through whichever map owns the value.
function nodeDot(status?: string): string {
  if (status === 'online' || status === 'offline' || status === 'never') {
    return dotClass(hostTone(status));
  }
  if (status && status in TOOL_TONE) {
    return dotClass(TOOL_TONE[status as ToolStatus]);
  }
  return dotClass('muted');
}

type HostData = { kind: 'host'; refId: string; name: string; iconUrl?: string; status?: string; count: number };
type SvcData = { kind: 'service'; refId: string; name: string; iconUrl?: string; status: Tool['status'] };

function HostNode({ data }: { data: HostData }) {
  return (
    <div className="flex items-center gap-2 rounded-card border-2 border-hairline-strong bg-surface-1 px-3 py-2">
      <Handle type="source" position={Position.Right} className="!border-0 !bg-transparent" />
      <EntityIcon url={data.iconUrl} name={data.name} size={28} />
      <div className="min-w-0">
        <p className="max-w-[150px] truncate text-sm font-medium text-content">{data.name}</p>
        <p className="text-[10px] text-muted">
          {data.count} service{data.count === 1 ? '' : 's'}
        </p>
      </div>
      <span className={`ml-1 h-2 w-2 shrink-0 rounded-full ${nodeDot(data.status)}`} />
    </div>
  );
}

function ServiceNode({ data }: { data: SvcData }) {
  return (
    <div className="flex items-center gap-2 rounded-button border border-hairline bg-canvas px-3 py-2">
      <Handle type="target" position={Position.Left} className="!border-0 !bg-transparent" />
      <EntityIcon url={data.iconUrl} name={data.name} size={20} />
      <span className="max-w-[130px] truncate text-xs text-content">{data.name}</span>
      <StatusPill status={data.status} />
    </div>
  );
}

const nodeTypes = { host: HostNode, service: ServiceNode };

// PortalGraph draws the server topology with React Flow: host nodes linked to
// their service nodes by angled (smoothstep) edges. Nodes are clickable.
export default function PortalGraph({
  tools,
  hosts,
  onOpenTool,
  onOpenHost,
}: {
  tools: Tool[];
  hosts: Host[];
  onOpenTool: (id: string) => void;
  onOpenHost?: (id: string) => void;
}) {
  const { nodes, edges } = useMemo(() => {
    const groups = groupToolsByHost(tools, hosts);
    const ns: Node[] = [];
    const es: Edge[] = [];
    const ROW = 88;
    const BLOCK_GAP = 48;
    let curY = 0;
    groups.forEach((g, gi) => {
      const hid = `h${gi}`;
      const count = g.tools.length;
      const blockH = Math.max(1, count) * ROW;
      ns.push({
        id: hid,
        type: 'host',
        position: { x: 0, y: curY + blockH / 2 - 28 },
        data: {
          kind: 'host',
          refId: g.host?.id ?? '',
          name: g.host?.name ?? 'Unassigned',
          iconUrl: g.host?.icon_url,
          status: g.host?.status,
          count,
        } satisfies HostData,
        draggable: false,
        selectable: false,
      });
      g.tools.forEach((t, k) => {
        const sid = `s${gi}_${k}`;
        ns.push({
          id: sid,
          type: 'service',
          position: { x: 340, y: curY + k * ROW },
          data: { kind: 'service', refId: t.id, name: t.name, iconUrl: t.icon_url, status: t.status } satisfies SvcData,
          draggable: false,
          selectable: false,
        });
        es.push({ id: `${hid}-${sid}`, source: hid, target: sid, type: 'smoothstep' });
      });
      curY += blockH + BLOCK_GAP;
    });
    return { nodes: ns, edges: es };
  }, [tools, hosts]);

  return (
    <div className="h-full w-full overflow-hidden rounded-card border border-hairline">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        colorMode="light"
        fitView
        fitViewOptions={{ padding: 0.2, maxZoom: 1 }}
        nodesDraggable={false}
        nodesConnectable={false}
        proOptions={{ hideAttribution: true }}
        defaultEdgeOptions={{ type: 'smoothstep', style: { stroke: '#bfbfc3', strokeWidth: 1.5 } }}
        onNodeClick={(_, node) => {
          const d = node.data as HostData | SvcData;
          if (d.kind === 'service') {
            onOpenTool(d.refId);
          } else if (d.refId) {
            onOpenHost?.(d.refId);
          }
        }}
      >
        <Background gap={22} size={1} color="#dcdcde" />
        <Controls showInteractive={false} />
      </ReactFlow>
    </div>
  );
}
