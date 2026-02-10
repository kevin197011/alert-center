import { Form, Input, Button, message } from 'antd';
import { UserOutlined, LockOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { authApi, type ApiResponse } from '../../services/api';
import { useAuthStore } from '../../store/auth';
import './login.css';

export default function Login() {
  const navigate = useNavigate();
  const { setAuth } = useAuthStore();
  const [form] = Form.useForm();

  const handleSubmit = async (values: { username: string; password: string }) => {
    try {
      const response = await authApi.login(values.username, values.password);
      const body = response.data as ApiResponse<{ user: { id: string; username: string; email: string; role: string }; token: string }>;
      const payload = body?.data ?? (response.data as { user?: unknown; token?: string });
      const token = payload?.token as string | undefined;
      const user = payload?.user as { id: string; username: string; email: string; role: string } | undefined;
      if (token && user) {
        setAuth(token, user);
        message.success('登录成功');
        navigate('/', { replace: true });
      } else {
        message.error('登录响应格式异常');
      }
    } catch (error: unknown) {
      const err = error as { response?: { data?: { message?: string } } };
      message.error(err?.response?.data?.message || '登录失败');
    }
  };

  return (
    <div className="login-page">
      <div className="login-orb login-orb--1" aria-hidden />
      <div className="login-orb login-orb--2" aria-hidden />
      <div className="login-orb login-orb--3" aria-hidden />

      <div className="login-card-wrap">
        <div className="login-card">
          <h1 className="login-title">Alert Center</h1>
          <p className="login-subtitle">告警管理平台 · 登录</p>
          <Form form={form} layout="vertical" onFinish={handleSubmit}>
            <Form.Item
              name="username"
              rules={[{ required: true, message: '请输入用户名' }]}
            >
              <Input prefix={<UserOutlined />} placeholder="用户名" size="large" autoComplete="username" />
            </Form.Item>
            <Form.Item
              name="password"
              rules={[{ required: true, message: '请输入密码' }]}
            >
              <Input.Password prefix={<LockOutlined />} placeholder="密码" size="large" autoComplete="current-password" />
            </Form.Item>
            <Form.Item>
              <Button type="primary" htmlType="submit" block size="large">
                登录
              </Button>
            </Form.Item>
          </Form>
        </div>
      </div>
    </div>
  );
}
