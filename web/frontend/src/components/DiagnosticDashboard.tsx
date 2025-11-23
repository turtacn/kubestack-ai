import React, { useState, useEffect } from 'react';
import { Table, Button, Modal, Form, Input, Select, Badge, Card, message } from 'antd';
import { PlayCircleOutlined } from '@ant-design/icons';
import { DiagnosisAPI, DiagnosisRequest } from '../services/api';

const { Option } = Select;

interface DiagnosisRecord {
  id: string;
  target: string;
  middleware: string;
  status: string;
  timestamp: string;
}

const DiagnosticDashboard: React.FC = () => {
  const [data, setData] = useState<DiagnosisRecord[]>([]);
  const [isModalVisible, setIsModalVisible] = useState(false);
  const [form] = Form.useForm();
  const [ws, setWs] = useState<WebSocket | null>(null);

  useEffect(() => {
    // In a real app, we would fetch initial list
    // setData([...])
  }, []);

  const handleTrigger = async (values: DiagnosisRequest) => {
    try {
      const res = await DiagnosisAPI.trigger(values);
      message.success(`Diagnosis started: ${res.id}`);
      setIsModalVisible(false);

      const newRecord: DiagnosisRecord = {
        id: res.id,
        target: values.target,
        middleware: values.middleware,
        status: 'Started',
        timestamp: new Date().toISOString(),
      };
      setData((prev) => [newRecord, ...prev]);

      // Connect WebSocket
      const socket = new WebSocket(`ws://localhost:8080/ws/diagnosis/${res.id}`);
      socket.onopen = () => {
        console.log('Connected to WS');
      };
      socket.onmessage = (event) => {
        const msg = JSON.parse(event.data);
        console.log('WS Message:', msg);
        if (msg.topic === res.id) {
            updateStatus(res.id, msg.payload.status);
        }
      };
      setWs(socket);

    } catch (error) {
      message.error('Failed to trigger diagnosis');
    }
  };

  const updateStatus = (id: string, status: string) => {
      setData(prev => prev.map(item => item.id === id ? { ...item, status } : item));
  };

  const columns = [
    { title: 'ID', dataIndex: 'id', key: 'id' },
    { title: 'Target', dataIndex: 'target', key: 'target' },
    { title: 'Middleware', dataIndex: 'middleware', key: 'middleware' },
    {
      title: 'Status',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => {
          let color = 'default';
          if (status === 'Completed') color = 'success';
          if (status === 'Failed' || status === 'Critical') color = 'error';
          if (status === 'InProgress' || status === 'Started') color = 'processing';
          return <Badge status={color as any} text={status} />;
      }
    },
    { title: 'Time', dataIndex: 'timestamp', key: 'timestamp' },
  ];

  return (
    <Card title="Diagnostic Dashboard" extra={<Button type="primary" icon={<PlayCircleOutlined />} onClick={() => setIsModalVisible(true)}>New Diagnosis</Button>}>
      <Table dataSource={data} columns={columns} rowKey="id" />

      <Modal title="New Diagnosis" open={isModalVisible} onCancel={() => setIsModalVisible(false)} onOk={() => form.submit()}>
        <Form form={form} onFinish={handleTrigger} layout="vertical">
          <Form.Item name="target" label="Target Cluster/Host" rules={[{ required: true }]}>
            <Input placeholder="e.g. redis-prod-01" />
          </Form.Item>
          <Form.Item name="middleware" label="Middleware Type" rules={[{ required: true }]}>
            <Select>
              <Option value="redis">Redis</Option>
              <Option value="mysql">MySQL</Option>
              <Option value="elasticsearch">Elasticsearch</Option>
              <Option value="kafka">Kafka</Option>
            </Select>
          </Form.Item>
          <Form.Item name="instance" label="Instance Name">
             <Input placeholder="e.g. redis-main" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
};

export default DiagnosticDashboard;
