import React, { useState, useContext } from 'react';
import {
  Paper,
  Typography,
  Box,
  Grid,
  TextField,
  Button,
  Divider,
  Alert,
  InputAdornment,
  IconButton,
  CircularProgress,
  Accordion,
  AccordionSummary,
  AccordionDetails
} from '@mui/material';
import ExpandMoreIcon from '@mui/icons-material/ExpandMore';
import Visibility from '@mui/icons-material/Visibility';
import VisibilityOff from '@mui/icons-material/VisibilityOff';
import AuthContext from '../context/AuthContext';

const Settings = () => {
  const { user, changePassword } = useContext(AuthContext);
  const [formData, setFormData] = useState({
    oldPassword: '',
    newPassword: '',
    confirmPassword: ''
  });
  const [showOldPassword, setShowOldPassword] = useState(false);
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [alert, setAlert] = useState({
    show: false,
    severity: 'info',
    message: ''
  });

  const { oldPassword, newPassword, confirmPassword } = formData;

  const handleChange = (e) => {
    setFormData({ ...formData, [e.target.name]: e.target.value });
    // Clear alerts when user types
    if (alert.show) {
      setAlert({ ...alert, show: false });
    }
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    
    // Validate passwords
    if (!oldPassword || !newPassword || !confirmPassword) {
      setAlert({
        show: true,
        severity: 'error',
        message: 'يرجى ملء جميع الحقول'
      });
      return;
    }

    if (newPassword !== confirmPassword) {
      setAlert({
        show: true,
        severity: 'error',
        message: 'كلمة المرور الجديدة وتأكيدها غير متطابقين'
      });
      return;
    }

    setLoading(true);
    try {
      const result = await changePassword(oldPassword, newPassword);
      
      if (result.success) {
        setAlert({
          show: true,
          severity: 'success',
          message: 'تم تغيير كلمة المرور بنجاح'
        });
        
        // Reset form
        setFormData({
          oldPassword: '',
          newPassword: '',
          confirmPassword: ''
        });
      } else {
        setAlert({
          show: true,
          severity: 'error',
          message: result.message || 'فشل تغيير كلمة المرور'
        });
      }
    } catch (error) {
      setAlert({
        show: true,
        severity: 'error',
        message: 'حدث خطأ أثناء تغيير كلمة المرور'
      });
    } finally {
      setLoading(false);
    }
  };

  return (
    <div>
      <Typography variant="h4" component="h1" gutterBottom>
        الإعدادات
      </Typography>
      <Divider sx={{ mb: 4 }} />

      <Grid container spacing={4}>
        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 3, mb: 3 }} elevation={2}>
            <Typography variant="h6" gutterBottom>
              معلومات المستخدم
            </Typography>
            <Box sx={{ mb: 3 }}>
              <TextField
                fullWidth
                label="اسم المستخدم"
                value={user?.username || ''}
                sx={{ mb: 2 }}
                disabled
              />
              <TextField
                fullWidth
                label="الدور"
                value={user?.role === 'admin' ? 'مسؤول' : 'مستخدم'}
                disabled
              />
            </Box>
          </Paper>

          <Paper sx={{ p: 3 }} elevation={2}>
            <Typography variant="h6" gutterBottom>
              تغيير كلمة المرور
            </Typography>
            
            {alert.show && (
              <Alert severity={alert.severity} sx={{ mb: 2 }}>
                {alert.message}
              </Alert>
            )}
            
            <Box component="form" onSubmit={handleSubmit}>
              <TextField
                fullWidth
                margin="normal"
                label="كلمة المرور الحالية"
                name="oldPassword"
                type={showOldPassword ? 'text' : 'password'}
                value={oldPassword}
                onChange={handleChange}
                required
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton
                        onClick={() => setShowOldPassword(!showOldPassword)}
                        edge="end"
                      >
                        {showOldPassword ? <VisibilityOff /> : <Visibility />}
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
              
              <TextField
                fullWidth
                margin="normal"
                label="كلمة المرور الجديدة"
                name="newPassword"
                type={showNewPassword ? 'text' : 'password'}
                value={newPassword}
                onChange={handleChange}
                required
                InputProps={{
                  endAdornment: (
                    <InputAdornment position="end">
                      <IconButton
                        onClick={() => setShowNewPassword(!showNewPassword)}
                        edge="end"
                      >
                        {showNewPassword ? <VisibilityOff /> : <Visibility />}
                      </IconButton>
                    </InputAdornment>
                  ),
                }}
              />
              
              <TextField
                fullWidth
                margin="normal"
                label="تأكيد كلمة المرور الجديدة"
                name="confirmPassword"
                type={showNewPassword ? 'text' : 'password'}
                value={confirmPassword}
                onChange={handleChange}
                required
              />
              
              <Button
                type="submit"
                variant="contained"
                color="primary"
                fullWidth
                sx={{ mt: 3 }}
                disabled={loading}
              >
                {loading ? <CircularProgress size={24} /> : 'تغيير كلمة المرور'}
              </Button>
            </Box>
          </Paper>
        </Grid>

        <Grid item xs={12} md={6}>
          <Paper sx={{ p: 3 }} elevation={2}>
            <Typography variant="h6" gutterBottom>
              إعدادات Evilginx2
            </Typography>
            
            <Accordion>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography>مسارات ملفات النظام</Typography>
              </AccordionSummary>
              <AccordionDetails>
                <TextField
                  fullWidth
                  margin="normal"
                  label="مسار قاعدة البيانات"
                  value={process.env.REACT_APP_DB_PATH || '~/.evilginx/data.db'}
                  disabled
                />
                <TextField
                  fullWidth
                  margin="normal"
                  label="مسار السجلات"
                  value={process.env.REACT_APP_LOGS_PATH || '~/.evilginx/logs'}
                  disabled
                />
                <Alert severity="info" sx={{ mt: 2 }}>
                  يمكن تغيير هذه المسارات عن طريق متغيرات البيئة في ملف .env
                </Alert>
              </AccordionDetails>
            </Accordion>
            
            <Accordion>
              <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                <Typography>معلومات حول النظام</Typography>
              </AccordionSummary>
              <AccordionDetails>
                <Typography variant="body2" paragraph>
                  لوحة تحكم Evilginx2 هي واجهة تحكم لنظام Evilginx2 لاختراق المواقع. يتم تطويرها كأداة تعليمية وعرض توضيحي فقط.
                </Typography>
                <Typography variant="body2" paragraph>
                  إخلاء المسؤولية: يجب استخدام هذه الأداة فقط للأغراض المشروعة مثل اختبار الاختراق المصرح به واختبار أمان النظام.
                </Typography>
                <Typography variant="body2">
                  الإصدار: 1.0.0
                </Typography>
              </AccordionDetails>
            </Accordion>
          </Paper>
        </Grid>
      </Grid>
    </div>
  );
};

export default Settings; 