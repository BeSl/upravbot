package screenshot

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"

	"github.com/cupbot/cupbot/internal/config"
)

var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	gdi32                      = syscall.NewLazyDLL("gdi32.dll")
	procGetSystemMetrics       = user32.NewProc("GetSystemMetrics")
	procGetDC                  = user32.NewProc("GetDC")
	procCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	procCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	procSelectObject           = gdi32.NewProc("SelectObject")
	procBitBlt                 = gdi32.NewProc("BitBlt")
	procGetDIBits              = gdi32.NewProc("GetDIBits")
	procDeleteObject           = gdi32.NewProc("DeleteObject")
	procDeleteDC               = gdi32.NewProc("DeleteDC")
	procReleaseDC              = user32.NewProc("ReleaseDC")
)

const (
	SM_CXSCREEN = 0
	SM_CYSCREEN = 1
	SRCCOPY     = 0x00CC0020
	BI_RGB      = 0
)

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

// Service provides screenshot functionality
type Service struct {
	config *config.Config
}

// NewService creates a new screenshot service
func NewService(cfg *config.Config) *Service {
	return &Service{
		config: cfg,
	}
}

// TakeScreenshot captures the desktop and saves it as an image file
func (s *Service) TakeScreenshot() (string, error) {
	if !s.config.Screenshot.Enabled {
		return "", fmt.Errorf("screenshot functionality is disabled")
	}

	// Get screen dimensions
	width, _, _ := procGetSystemMetrics.Call(SM_CXSCREEN)
	height, _, _ := procGetSystemMetrics.Call(SM_CYSCREEN)
	screenWidth := int(width)
	screenHeight := int(height)

	// Apply size limits from configuration
	if s.config.Screenshot.MaxWidth > 0 && screenWidth > s.config.Screenshot.MaxWidth {
		screenHeight = screenHeight * s.config.Screenshot.MaxWidth / screenWidth
		screenWidth = s.config.Screenshot.MaxWidth
	}
	if s.config.Screenshot.MaxHeight > 0 && screenHeight > s.config.Screenshot.MaxHeight {
		screenWidth = screenWidth * s.config.Screenshot.MaxHeight / screenHeight
		screenHeight = s.config.Screenshot.MaxHeight
	}

	// Create bitmap and capture screen
	bitmap, err := s.captureScreen(screenWidth, screenHeight)
	if err != nil {
		return "", fmt.Errorf("failed to capture screen: %w", err)
	}

	// Create storage directory if it doesn't exist
	storageDir := s.config.Screenshot.StoragePath
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Generate filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("screenshot_%s.%s", timestamp, s.config.Screenshot.Format)
	filepath := filepath.Join(storageDir, filename)

	// Save image
	if err := s.saveImage(bitmap, filepath, screenWidth, screenHeight); err != nil {
		return "", fmt.Errorf("failed to save screenshot: %w", err)
	}

	return filepath, nil
}

// captureScreen captures the screen and returns the bitmap data
func (s *Service) captureScreen(width, height int) ([]byte, error) {
	// Get the device context of the screen
	hdcScreen, _, _ := procGetDC.Call(0)
	if hdcScreen == 0 {
		return nil, fmt.Errorf("failed to get screen DC")
	}
	defer procReleaseDC.Call(0, hdcScreen)

	// Create a compatible device context
	hdcMem, _, _ := procCreateCompatibleDC.Call(hdcScreen)
	if hdcMem == 0 {
		return nil, fmt.Errorf("failed to create compatible DC")
	}
	defer procDeleteDC.Call(hdcMem)

	// Create a compatible bitmap
	hbmScreen, _, _ := procCreateCompatibleBitmap.Call(hdcScreen, uintptr(width), uintptr(height))
	if hbmScreen == 0 {
		return nil, fmt.Errorf("failed to create compatible bitmap")
	}
	defer procDeleteObject.Call(hbmScreen)

	// Select the bitmap into the memory device context
	procSelectObject.Call(hdcMem, hbmScreen)

	// Copy the screen to the memory device context
	success, _, _ := procBitBlt.Call(
		hdcMem, 0, 0, uintptr(width), uintptr(height),
		hdcScreen, 0, 0, SRCCOPY)
	if success == 0 {
		return nil, fmt.Errorf("failed to copy screen")
	}

	// Prepare bitmap info structure
	bmi := BITMAPINFOHEADER{
		BiSize:        40,
		BiWidth:       int32(width),
		BiHeight:      -int32(height), // Negative for top-down bitmap
		BiPlanes:      1,
		BiBitCount:    24, // 24-bit RGB
		BiCompression: BI_RGB,
	}

	// Calculate bitmap size
	stride := ((width*3 + 3) / 4) * 4 // 4-byte alignment
	bitmapSize := stride * height
	bitmap := make([]byte, bitmapSize)

	// Get bitmap bits
	ret, _, _ := procGetDIBits.Call(
		hdcScreen,
		hbmScreen,
		0,
		uintptr(height),
		uintptr(unsafe.Pointer(&bitmap[0])),
		uintptr(unsafe.Pointer(&bmi)),
		0) // DIB_RGB_COLORS = 0
	if ret == 0 {
		return nil, fmt.Errorf("failed to get bitmap bits")
	}

	return bitmap, nil
}

// saveImage saves the bitmap data as an image file
func (s *Service) saveImage(bitmap []byte, filepath string, width, height int) error {
	// Convert bitmap to image.Image
	img := s.bitmapToImage(bitmap, width, height)

	// Create file
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Save based on format
	switch s.config.Screenshot.Format {
	case "jpg", "jpeg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: s.config.Screenshot.Quality})
	case "png":
		return png.Encode(file, img)
	default:
		return fmt.Errorf("unsupported image format: %s", s.config.Screenshot.Format)
	}
}

// bitmapToImage converts bitmap data to image.Image
func (s *Service) bitmapToImage(bitmap []byte, width, height int) image.Image {
	stride := ((width*3 + 3) / 4) * 4
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			bitmapOffset := y*stride + x*3
			imageOffset := (y*width + x) * 4

			// Bitmap is BGR, convert to RGBA
			if bitmapOffset+2 < len(bitmap) {
				img.Pix[imageOffset+0] = bitmap[bitmapOffset+2] // R
				img.Pix[imageOffset+1] = bitmap[bitmapOffset+1] // G
				img.Pix[imageOffset+2] = bitmap[bitmapOffset+0] // B
				img.Pix[imageOffset+3] = 255                    // A
			}
		}
	}

	return img
}

// GetScreenshotList returns a list of available screenshots
func (s *Service) GetScreenshotList() ([]ScreenshotInfo, error) {
	storageDir := s.config.Screenshot.StoragePath

	entries, err := os.ReadDir(storageDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ScreenshotInfo{}, nil // Empty list if directory doesn't exist
		}
		return nil, fmt.Errorf("failed to read screenshot directory: %w", err)
	}

	var screenshots []ScreenshotInfo
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		screenshot := ScreenshotInfo{
			Name:    info.Name(),
			Path:    filepath.Join(storageDir, info.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		}
		screenshots = append(screenshots, screenshot)
	}

	return screenshots, nil
}

// DeleteScreenshot deletes a screenshot file
func (s *Service) DeleteScreenshot(filename string) error {
	// Validate filename to prevent path traversal
	if filepath.Dir(filename) != "." {
		return fmt.Errorf("invalid filename")
	}

	filePath := filepath.Join(s.config.Screenshot.StoragePath, filename)
	return os.Remove(filePath)
}

// ScreenshotInfo contains information about a screenshot file
type ScreenshotInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}
