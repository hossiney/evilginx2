import React from 'react';
import { Link } from 'react-router-dom';
import { Box, Typography, Button, Paper } from '@mui/material';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HomeIcon from '@mui/icons-material/Home';

const NotFound = () => {
  return (
    <Box
      sx={{
        display: 'flex',
        justifyContent: 'center',
        alignItems: 'center',
        height: '100vh',
        backgroundColor: '#f5f5f5'
      }}
    >
      <Paper
        elevation={3}
        sx={{
          p: 5,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          maxWidth: 500,
          textAlign: 'center'
        }}
      >
        <ErrorOutlineIcon sx={{ fontSize: 80, color: 'error.main', mb: 2 }} />
        <Typography variant="h4" component="h1" gutterBottom>
          404 - الصفحة غير موجودة
        </Typography>
        <Typography variant="body1" color="text.secondary" paragraph>
          عذراً، الصفحة التي تبحث عنها غير موجودة أو تم نقلها أو إزالتها.
        </Typography>
        <Button
          component={Link}
          to="/"
          variant="contained"
          color="primary"
          startIcon={<HomeIcon />}
          sx={{ mt: 2 }}
        >
          العودة إلى الصفحة الرئيسية
        </Button>
      </Paper>
    </Box>
  );
};

export default NotFound;
