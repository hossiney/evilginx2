import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Paper,
  Typography,
  Box,
  Grid,
  TableContainer,
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableCell,
  CircularProgress,
  Alert,
  Chip,
  TextField,
  MenuItem,
  InputAdornment,
  IconButton,
  Button,
  Divider
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import FilterListIcon from '@mui/icons-material/FilterList';
import ClearIcon from '@mui/icons-material/Clear';
import moment from 'moment';

const Logs = () => {
  const [logs, setLogs] = useState([]);
  const [filteredLogs, setFilteredLogs] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');
  const [showFilters, setShowFilters] = useState(false);
  const [filters, setFilters] = useState({
    sessionId: '',
    startDate: '',
    endDate: ''
  });
  const [sessionIds, setSessionIds] = useState([]);

  useEffect(() => {
    const fetchLogs = async () => {
      try {
        const res = await axios.get('http://5.199.168.182:5000/api/logs');
        setLogs(res.data.data);
        setFilteredLogs(res.data.data);
        
        // Extract unique session IDs
        const uniqueSessionIds = [...new Set(res.data.data.map(log => log.sessionId))];
        setSessionIds(uniqueSessionIds);
        
        setLoading(false);
      } catch (err) {
        setError(err.response?.data?.message || 'خطأ في جلب السجلات');
        setLoading(false);
      }
    };

    fetchLogs();
  }, []);

  useEffect(() => {
    // Apply client-side search filtering
    if (searchTerm.trim() === '') {
      setFilteredLogs(logs);
    } else {
      const term = searchTerm.toLowerCase();
      const filtered = logs.filter(
        log => 
          log.message?.toLowerCase().includes(term) ||
          log.level?.toLowerCase().includes(term) ||
          log.timestamp?.toLowerCase().includes(term) ||
          log.sessionId?.toLowerCase().includes(term)
      );
      setFilteredLogs(filtered);
    }
  }, [searchTerm, logs]);

  const handleFilterChange = (e) => {
    setFilters({ ...filters, [e.target.name]: e.target.value });
  };

  const applyFilters = async () => {
    setLoading(true);
    
    try {
      // If filters are empty, reset to all logs
      if (!filters.sessionId && !filters.startDate && !filters.endDate) {
        setFilteredLogs(logs);
        setLoading(false);
        return;
      }
      
      // Otherwise, apply filters through API
      const res = await axios.post('http://5.199.168.182:5000/api/logs/filter', filters);
      setFilteredLogs(res.data.data);
      // Reset search term when applying filters
      setSearchTerm('');
    } catch (err) {
      setError(err.response?.data?.message || 'خطأ في تطبيق الفلتر');
    } finally {
      setLoading(false);
    }
  };

  const resetFilters = () => {
    setFilters({
      sessionId: '',
      startDate: '',
      endDate: ''
    });
    setFilteredLogs(logs);
    setSearchTerm('');
  };

  const getLevelColor = (level) => {
    switch (level) {
      case 'err':
      case 'error':
        return 'error';
      case 'war':
      case 'warning':
        return 'warning';
      case 'success':
      case '+++':
        return 'success';
      case 'inf':
      case 'info':
        return 'info';
      case 'dbg':
        return 'default';
      default:
        return 'default';
    }
  };

  if (loading && logs.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', py: 5 }}>
        <CircularProgress />
      </Box>
    );
  }

  if (error && logs.length === 0) {
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
          سجلات النظام
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

      <Divider sx={{ mb: 3 }} />

      {/* Search */}
      <TextField
        fullWidth
        variant="outlined"
        placeholder="بحث في السجلات..."
        value={searchTerm}
        onChange={(e) => setSearchTerm(e.target.value)}
        sx={{ mb: 3 }}
        InputProps={{
          startAdornment: (
            <InputAdornment position="start">
              <SearchIcon />
            </InputAdornment>
          ),
          endAdornment: searchTerm ? (
            <InputAdornment position="end">
              <IconButton size="small" onClick={() => setSearchTerm('')}>
                <ClearIcon fontSize="small" />
              </IconButton>
            </InputAdornment>
          ) : null
        }}
      />

      {/* Filters */}
      {showFilters && (
        <Paper sx={{ p: 3, mb: 3 }} elevation={2}>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} sm={3}>
              <TextField
                select
                label="معرف الجلسة"
                name="sessionId"
                fullWidth
                value={filters.sessionId}
                onChange={handleFilterChange}
                variant="outlined"
                InputProps={{
                  endAdornment: filters.sessionId ? (
                    <InputAdornment position="end">
                      <IconButton size="small" onClick={() => setFilters({...filters, sessionId: ''})}>
                        <ClearIcon fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  ) : null
                }}
              >
                <MenuItem value="">الكل</MenuItem>
                {sessionIds.map(sid => (
                  <MenuItem key={sid} value={sid}>{sid}</MenuItem>
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

      {/* Logs Table */}
      {filteredLogs.length > 0 ? (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>التاريخ والوقت</TableCell>
                <TableCell>معرف الجلسة</TableCell>
                <TableCell>المستوى</TableCell>
                <TableCell>الرسالة</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredLogs.map((log, index) => (
                <TableRow key={index}>
                  <TableCell>{log.timestamp}</TableCell>
                  <TableCell>
                    <Chip
                      label={log.sessionId}
                      size="small"
                      variant="outlined"
                      color="primary"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={log.level}
                      size="small"
                      color={getLevelColor(log.level)}
                    />
                  </TableCell>
                  <TableCell sx={{ maxWidth: '500px', wordBreak: 'break-word' }}>
                    {log.message}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      ) : (
        <Alert severity="info">لا توجد سجلات مطابقة للبحث أو الفلتر</Alert>
      )}
    </div>
  );
};

export default Logs; 