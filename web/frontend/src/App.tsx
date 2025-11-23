import React from 'react';
import './App.css';
import DiagnosticDashboard from './components/DiagnosticDashboard';
import ExecutionPlanViewer from './components/ExecutionPlanViewer';
import Login from './components/Login';
import { Layout, Menu, Typography } from 'antd';
import { DashboardOutlined, PartitionOutlined, LogoutOutlined } from '@ant-design/icons';
import { BrowserRouter as Router, Route, Routes, Link, useLocation, useNavigate } from 'react-router-dom';
import { useEffect } from 'react';

const { Header, Content, Footer } = Layout;

// Wrapper to handle layout logic based on route
const AppLayout: React.FC<{ children: React.ReactNode }> = ({ children }) => {
    const location = useLocation();
    const navigate = useNavigate();
    const isLoginPage = location.pathname === '/login';

    useEffect(() => {
        const token = localStorage.getItem('token');
        if (!token && !isLoginPage) {
            navigate('/login');
        }
    }, [location, navigate, isLoginPage]);

    const handleLogout = () => {
        localStorage.removeItem('token');
        navigate('/login');
    };

    if (isLoginPage) {
        return <>{children}</>;
    }

    return (
        <Layout className="layout" style={{ minHeight: '100vh' }}>
            <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                <div style={{ display: 'flex', alignItems: 'center' }}>
                    <div className="logo" style={{ color: 'white', marginRight: '20px', fontWeight: 'bold', fontSize: '18px' }}>
                        KubeStack-AI
                    </div>
                    <Menu theme="dark" mode="horizontal" defaultSelectedKeys={[location.pathname]} selectedKeys={[location.pathname]}>
                        <Menu.Item key="/" icon={<DashboardOutlined />}>
                            <Link to="/">Dashboard</Link>
                        </Menu.Item>
                        <Menu.Item key="/plans" icon={<PartitionOutlined />}>
                            <Link to="/plans">Execution Plans</Link>
                        </Menu.Item>
                    </Menu>
                </div>
                 <div style={{ color: 'white', cursor: 'pointer' }} onClick={handleLogout}>
                    <LogoutOutlined /> Logout
                </div>
            </Header>
            <Content style={{ padding: '0 50px', marginTop: 24 }}>
                <div className="site-layout-content" style={{ background: '#fff', padding: 24, minHeight: 280 }}>
                    {children}
                </div>
            </Content>
            <Footer style={{ textAlign: 'center' }}>KubeStack-AI ©2024</Footer>
        </Layout>
    );
};

function App() {
  return (
    <Router>
        <AppLayout>
             <Routes>
                <Route path="/login" element={<Login />} />
                <Route path="/" element={<DiagnosticDashboard />} />
                <Route path="/plans" element={<ExecutionPlanViewer />} />
             </Routes>
        </AppLayout>
    </Router>
  );
}

export default App;
