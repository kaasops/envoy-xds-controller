import React, { useState } from 'react'
import { 
  IconButton, 
  Menu, 
  MenuItem, 
  Dialog, 
  DialogTitle, 
  DialogContent, 
  DialogContentText, 
  DialogActions, 
  Button,
  Typography,
  Box,
  CircularProgress
} from '@mui/material'
import HelpOutlineIcon from '@mui/icons-material/HelpOutline'
import { useBuildInfo } from '../../api/hooks/useBuildInfo'

const HelpMenu: React.FC = () => {
  const [anchorEl, setAnchorEl] = useState<null | HTMLElement>(null)
  const [aboutDialogOpen, setAboutDialogOpen] = useState(false)
  const { data: buildInfo, isLoading, isError } = useBuildInfo()
  
  const handleMenuOpen = (event: React.MouseEvent<HTMLElement>) => {
    setAnchorEl(event.currentTarget)
  }
  
  const handleMenuClose = () => {
    setAnchorEl(null)
  }
  
  const handleAboutClick = () => {
    handleMenuClose()
    setAboutDialogOpen(true)
  }
  
  const handleHelpClick = () => {
    handleMenuClose()
    // Open help documentation in a new tab
    window.open('https://github.com/kaasops/envoy-xds-controller/blob/main/README.md', '_blank')
  }
  
  const handleAboutDialogClose = () => {
    setAboutDialogOpen(false)
  }
  
  return (
    <>
      <IconButton
        color="inherit"
        aria-label="help menu"
        onClick={handleMenuOpen}
        size="small"
      >
        <HelpOutlineIcon />
      </IconButton>
      
      <Menu
        anchorEl={anchorEl}
        open={Boolean(anchorEl)}
        onClose={handleMenuClose}
      >
        <MenuItem onClick={handleAboutClick}>About</MenuItem>
        <MenuItem onClick={handleHelpClick}>Help</MenuItem>
      </Menu>
      
      <Dialog
        open={aboutDialogOpen}
        onClose={handleAboutDialogClose}
        aria-labelledby="about-dialog-title"
      >
        <DialogTitle id="about-dialog-title">
          About Envoy XDS Controller
        </DialogTitle>
        <DialogContent>
          {isLoading ? (
            <Box display="flex" justifyContent="center" p={2}>
              <CircularProgress />
            </Box>
          ) : isError ? (
            <DialogContentText color="error">
              Failed to load version information.
            </DialogContentText>
          ) : buildInfo ? (
            <Box>
              <Typography variant="body1" gutterBottom>
                <strong>Version:</strong> {buildInfo.version}
              </Typography>
              <Typography variant="body1" gutterBottom>
                <strong>Commit Hash:</strong> {buildInfo.commitHash}
              </Typography>
              <Typography variant="body1" gutterBottom>
                <strong>Build Date:</strong> {buildInfo.buildDate}
              </Typography>
            </Box>
          ) : (
            <DialogContentText>
              No version information available.
            </DialogContentText>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={handleAboutDialogClose} color="primary">
            Close
          </Button>
        </DialogActions>
      </Dialog>
    </>
  )
}

export default HelpMenu