package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// IsFileExist 检查文件是否存在
func IsFileExist(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// CreateDirIfNotExist 如果目录不存在则创建
func CreateDirIfNotExist(path string) error {
	if !IsFileExist(path) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

// CopyFile 复制文件
func CopyFile(src, dst string) error {
	// 确保目标目录存在
	dstDir := filepath.Dir(dst)
	if err := CreateDirIfNotExist(dstDir); err != nil {
		return fmt.Errorf("创建目标目录失败: %v", err)
	}

	// 打开源文件
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %v", err)
	}
	defer sourceFile.Close()

	// 创建目标文件
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %v", err)
	}
	defer destFile.Close()

	// 复制内容
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("复制文件内容失败: %v", err)
	}

	// 同步到磁盘
	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("同步文件到磁盘失败: %v", err)
	}

	// 复制文件权限
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %v", err)
	}

	err = os.Chmod(dst, sourceInfo.Mode())
	if err != nil {
		return fmt.Errorf("设置目标文件权限失败: %v", err)
	}

	return nil
}

// CopyDir 复制目录
func CopyDir(src, dst string) error {
	// 获取源目录信息
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("获取源目录信息失败: %v", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("源路径不是目录")
	}

	// 创建目标目录
	if err := CreateDirIfNotExist(dst); err != nil {
		return err
	}

	// 获取源目录中的文件和子目录
	entries, err := os.ReadDir(src)
	if err != nil {
		return fmt.Errorf("读取源目录内容失败: %v", err)
	}

	// 遍历所有文件和子目录
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// 递归复制子目录
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// 复制文件
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// ExtractTarGz 解压tar.gz文件
func ExtractTarGz(tarGzFile, destDir string) error {
	// 打开tar.gz文件
	file, err := os.Open(tarGzFile)
	if err != nil {
		return fmt.Errorf("打开tar.gz文件失败: %v", err)
	}
	defer file.Close()

	// 创建gzip读取器
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("创建gzip读取器失败: %v", err)
	}
	defer gzReader.Close()

	// 创建tar读取器
	tarReader := tar.NewReader(gzReader)

	// 确保目标目录存在
	if err := CreateDirIfNotExist(destDir); err != nil {
		return err
	}

	// 遍历tar文件中的所有文件
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break // 文件结束
		}
		if err != nil {
			return fmt.Errorf("读取tar文件项失败: %v", err)
		}

		// 构建完整路径
		path := filepath.Join(destDir, header.Name)

		// 检查路径是否在目标目录范围内（防止路径遍历攻击）
		if !strings.HasPrefix(path, destDir) {
			return fmt.Errorf("不安全的路径: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			// 创建目录
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %v", err)
			}
		case tar.TypeReg:
			// 创建文件
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("创建父目录失败: %v", err)
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("创建文件失败: %v", err)
			}

			if _, err := io.Copy(file, tarReader); err != nil {
				file.Close()
				return fmt.Errorf("写入文件内容失败: %v", err)
			}

			file.Close()
		}
	}

	return nil
}

// ExtractZip 解压zip文件
func ExtractZip(zipFile, destDir string) error {
	// 打开zip文件
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("打开zip文件失败: %v", err)
	}
	defer reader.Close()

	// 确保目标目录存在
	if err := CreateDirIfNotExist(destDir); err != nil {
		return err
	}

	// 遍历zip文件中的所有文件
	for _, file := range reader.File {
		// 构建完整路径
		path := filepath.Join(destDir, file.Name)

		// 检查路径是否在目标目录范围内（防止路径遍历攻击）
		if !strings.HasPrefix(path, destDir) {
			return fmt.Errorf("不安全的路径: %s", file.Name)
		}

		if file.FileInfo().IsDir() {
			// 创建目录
			if err := os.MkdirAll(path, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %v", err)
			}
			continue
		}

		// 创建父目录
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建父目录失败: %v", err)
		}

		// 打开zip中的文件
		fileReader, err := file.Open()
		if err != nil {
			return fmt.Errorf("打开zip中的文件失败: %v", err)
		}

		// 创建目标文件
		targetFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, file.Mode())
		if err != nil {
			fileReader.Close()
			return fmt.Errorf("创建目标文件失败: %v", err)
		}

		// 复制内容
		if _, err := io.Copy(targetFile, fileReader); err != nil {
			fileReader.Close()
			targetFile.Close()
			return fmt.Errorf("写入文件内容失败: %v", err)
		}

		// 关闭文件
		fileReader.Close()
		targetFile.Close()
	}

	return nil
}

// WriteFile 写入内容到文件
func WriteFile(path string, content []byte, perm os.FileMode) error {
	// 确保目标目录存在
	dir := filepath.Dir(path)
	if err := CreateDirIfNotExist(dir); err != nil {
		return err
	}

	return os.WriteFile(path, content, perm)
}

// ReadFile 从文件读取内容
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}
