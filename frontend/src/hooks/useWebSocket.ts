import { useEffect, useState, useCallback, useRef } from 'react';
import { message } from 'antd';

interface AlertMessage {
  type: string;
  alert_id: string;
  rule_id: string;
  rule_name: string;
  severity: string;
  status: string;
  labels: Record<string, unknown>;
  message: string;
  timestamp: string;
}

interface SLABreachMessage {
  type: string;
  breach_id: string;
  alert_id: string;
  severity: string;
  breach_type: string;
  timestamp: string;
}

interface TicketMessage {
  type: string;
  ticket_id: string;
  title: string;
  status: string;
  action: string;
  timestamp: string;
}

interface UseWebSocketOptions {
  onAlert?: (alert: AlertMessage) => void;
  onSLABreach?: (breach: SLABreachMessage) => void;
  onTicket?: (ticket: TicketMessage) => void;
}

export function useWebSocket(options: UseWebSocketOptions = {}) {
  const [alerts, setAlerts] = useState<AlertMessage[]>([]);
  const [slaBreaches, setSLABreaches] = useState<SLABreachMessage[]>([]);
  const [tickets, setTickets] = useState<TicketMessage[]>([]);
  const [connected, setConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const optionsRef = useRef(options);
  optionsRef.current = options;

  const connect = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/api/v1/ws`;

    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      wsRef.current = new WebSocket(wsUrl);

      wsRef.current.onopen = () => {
        console.log('WebSocket connected');
        setConnected(true);
      };

      wsRef.current.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          const opts = optionsRef.current;
          switch (data.type) {
            case 'alert':
              const alert: AlertMessage = data;
              setAlerts((prev) => [alert, ...prev].slice(0, 100));
              message.info(`新告警: ${alert.rule_name} - ${alert.severity}`);
              opts.onAlert?.(alert);
              break;
            case 'sla_breach':
              const breach: SLABreachMessage = data;
              setSLABreaches((prev) => [breach, ...prev].slice(0, 50));
              message.warning(`SLA违约: ${breach.breach_type} - ${breach.severity}`);
              opts.onSLABreach?.(breach);
              break;
            case 'ticket':
              const ticket: TicketMessage = data;
              setTickets((prev) => [ticket, ...prev].slice(0, 50));
              message.info(`工单更新: ${ticket.title} - ${ticket.action}`);
              opts.onTicket?.(ticket);
              break;
            default:
              console.log('Unknown message type:', data.type);
          }
        } catch (error) {
          console.error('Failed to parse WebSocket message:', error);
        }
      };

      wsRef.current.onclose = () => {
        console.log('WebSocket disconnected');
        setConnected(false);
        reconnectTimeoutRef.current = setTimeout(() => {
          connect();
        }, 5000);
      };

      wsRef.current.onerror = () => {
        // Log suppressed to avoid console spam; onclose will trigger reconnect
      };
    } catch (error) {
      console.error('Failed to create WebSocket:', error);
      reconnectTimeoutRef.current = setTimeout(() => {
        connect();
      }, 5000);
    }
  }, []);

  useEffect(() => {
    connect();
    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
        reconnectTimeoutRef.current = null;
      }
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, [connect]);

  const clearAlerts = () => {
    setAlerts([]);
  };

  const clearSLABreaches = () => {
    setSLABreaches([]);
  };

  const clearTickets = () => {
    setTickets([]);
  };

  const removeAlert = (alertId: string) => {
    setAlerts((prev) => prev.filter((a) => a.alert_id !== alertId));
  };

  return {
    alerts,
    slaBreaches,
    tickets,
    connected,
    clearAlerts,
    clearSLABreaches,
    clearTickets,
    removeAlert,
    alertCount: alerts.length,
    slaBreachCount: slaBreaches.length,
    ticketCount: tickets.length,
  };
}
