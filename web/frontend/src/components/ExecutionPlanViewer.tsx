import React, { useState } from 'react';
import ReactFlow, { Background, Controls, Node, Edge } from 'reactflow';
import 'reactflow/dist/style.css';
import { Button, Drawer, Descriptions, Card, Badge } from 'antd';

const initialNodes: Node[] = [
  { id: '1', position: { x: 100, y: 100 }, data: { label: 'Restart Redis' }, type: 'input' },
  { id: '2', position: { x: 100, y: 200 }, data: { label: 'Check Health' } },
];
const initialEdges: Edge[] = [{ id: 'e1-2', source: '1', target: '2' }];

const ExecutionPlanViewer: React.FC = () => {
  const [open, setOpen] = useState(false);

  return (
    <Card title="Execution Plan Viewer" style={{ height: 600 }}>
       <div style={{ height: 500 }}>
        <ReactFlow nodes={initialNodes} edges={initialEdges}>
            <Background />
            <Controls />
        </ReactFlow>
       </div>
       <div style={{ marginTop: 16 }}>
           <Button type="primary" onClick={() => setOpen(true)}>Review Plan</Button>
       </div>

       <Drawer title="Plan Details" placement="right" onClose={() => setOpen(false)} open={open}>
            <Descriptions column={1} bordered>
                <Descriptions.Item label="Risk Score"><Badge status="warning" text="Medium" /></Descriptions.Item>
                <Descriptions.Item label="Actions">2 Actions</Descriptions.Item>
                <Descriptions.Item label="Rollback">Supported</Descriptions.Item>
            </Descriptions>
            <Button type="primary" block style={{ marginTop: 16 }}>Approve & Execute</Button>
       </Drawer>
    </Card>
  );
};

export default ExecutionPlanViewer;
