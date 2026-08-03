const FilemanagerModule = {
    currentPath: '/sdcard',
    fileList: [],
    editingFilePath: '',
    editorContent: '',
    newDirName: '',
    newFileName: '',
    renameOldPath: '',
    renameNewName: '',
    showEditorModal: false,
    showUploadModal: false,
    showDirModal: false,
    showFileModal: false,
    showRenameModal: false,
    fileManagerShortcuts: [],
    showAddShortcutModal: false,
    newShortcutName: '',
    newShortcutPath: '',
    isDragging: false,

    // New State Variables
    storageInfo: { total_str: '0 MB', free_str: '0 MB', used_str: '0 MB', used_pct: 0, percent: 0, mount: '/sdcard' },
    fmClipboard: { action: null, path: '', name: '' },
    fmSearchQuery: '',
    fmSelectedPaths: [],
    showPermModal: false,
    showImagePreviewModal: false,
    permTarget: { path: '', name: '', is_dir: false },
    permMode: '0755',
    permOwner: 'root:root',
    permRecursive: false,
    imagePreview: { path: '', name: '', url: '', loading: false, error: false, size_str: '', width: 0, height: 0 },

    get currentPathSegments() {
        return this.currentPath.split('/').filter(Boolean);
    },

    getParentPath(path) {
        if (!path || path === '/') return '/';
        let clean = path;
        if (clean.endsWith('/') && clean.length > 1) {
            clean = clean.slice(0, -1);
        }
        const idx = clean.lastIndexOf('/');
        if (idx <= 0) return '/';
        return clean.substring(0, idx);
    },

    async fetchStorageInfo() {
        try {
            const res = await fetch('/api/files/storage');
            if (res.ok) {
                const data = await res.json();
                const formatSize = (bytes) => {
                    if (bytes === 0) return '0 B';
                    const k = 1024;
                    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
                    const i = Math.floor(Math.log(bytes) / Math.log(k));
                    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
                };
                this.storageInfo = {
                    total_str: formatSize(data.total || 0),
                    free_str: formatSize(data.free || 0),
                    used_str: formatSize(data.used || 0),
                    used_pct: data.used_pct || 0,
                    percent: Math.round(data.used_pct || 0),
                    mount: this.currentPath
                };
            }
        } catch (e) {}
    },

    async fetchFileList(path) {
        try {
            this.fmClearSelection();
            const res = await fetch('/api/files/list?path=' + encodeURIComponent(path || ''));
            if (res.ok) {
                const data = await res.json();
                this.currentPath = data.path;
                let files = data.files || [];
                if (this.currentPath !== '/') {
                    const parent = this.getParentPath(this.currentPath);
                    files.unshift({
                        name: '..',
                        path: parent,
                        is_dir: true,
                        permissions: 'd--r--r--',
                        size: 0,
                        is_parent: true
                    });
                }
                this.fileList = files;
                this.fetchStorageInfo();
            }
        } catch (e) {}
    },

    navigateBreadcrumb(index) {
        const segments = this.currentPathSegments.slice(0, index + 1);
        const targetPath = '/' + segments.join('/');
        this.fetchFileList(targetPath);
    },

    // Clipboard helpers
    fmCut(file) {
        if (!file || file.is_parent) return;
        this.fmClipboard = { action: 'cut', path: file.path, name: file.name };
        this.showToast && this.showToast('Clipboard', `Cut "${file.name}" to clipboard`, 'info');
    },

    fmCopy(file) {
        if (!file || file.is_parent) return;
        this.fmClipboard = { action: 'copy', path: file.path, name: file.name };
        this.showToast && this.showToast('Clipboard', `Copied "${file.name}" to clipboard`, 'info');
    },

    fmClearClipboard() {
        this.fmClipboard = { action: null, path: '', name: '' };
    },

    async fmPaste() {
        if (!this.fmClipboard.action || !this.fmClipboard.path) return;
        const targetPath = this.currentPath + (this.currentPath.endsWith('/') ? '' : '/') + this.fmClipboard.name;
        const endpoint = this.fmClipboard.action === 'cut' ? '/api/files/move' : '/api/files/copy';

        try {
            const res = await fetch(endpoint, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ src: this.fmClipboard.path, dst: targetPath })
            });
            const data = await res.json();
            if (data.success) {
                this.showToast && this.showToast('Paste Success', `Pasted "${this.fmClipboard.name}" successfully`, 'success');
                if (this.fmClipboard.action === 'cut') {
                    this.fmClearClipboard();
                }
                this.fetchFileList(this.currentPath);
            } else {
                this.showToast && this.showToast('Paste Error', data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Paste Error', 'Paste operation failed', 'error');
        }
    },

    // Selection helpers
    fmToggleSelect(path) {
        const idx = this.fmSelectedPaths.indexOf(path);
        if (idx > -1) {
            this.fmSelectedPaths.splice(idx, 1);
        } else {
            this.fmSelectedPaths.push(path);
        }
    },

    fmToggleAll() {
        const selectable = this.fileList.filter(f => !f.is_parent).map(f => f.path);
        if (this.fmAllSelected()) {
            this.fmSelectedPaths = [];
        } else {
            this.fmSelectedPaths = [...selectable];
        }
    },

    fmAllSelected() {
        const selectable = this.fileList.filter(f => !f.is_parent);
        return selectable.length > 0 && this.fmSelectedPaths.length === selectable.length;
    },

    fmClearSelection() {
        this.fmSelectedPaths = [];
    },

    // Batch Action helpers
    async fmDeleteSelected() {
        if (this.fmSelectedPaths.length === 0) return;
        const count = this.fmSelectedPaths.length;
        this.showConfirm(
            'Batch Delete',
            `Are you sure you want to delete ${count} selected item(s)?`,
            async () => {
                try {
                    const res = await fetch('/api/files/batch', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ action: 'delete', paths: this.fmSelectedPaths })
                    });
                    const data = await res.json();
                    if (data.success) {
                        this.showToast && this.showToast('Batch Delete', `Deleted ${count} item(s)`, 'success');
                        this.fmClearSelection();
                        this.fetchFileList(this.currentPath);
                    } else {
                        this.showToast && this.showToast('Batch Delete Error', data.error, 'error');
                    }
                } catch (e) {
                    this.showToast && this.showToast('Batch Delete Error', 'Batch delete failed', 'error');
                }
            }
        );
    },

    async fmCopySelected() {
        if (this.fmSelectedPaths.length === 0) return;
        try {
            const res = await fetch('/api/files/batch', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: 'copy', paths: this.fmSelectedPaths, dest_dir: this.currentPath })
            });
            const data = await res.json();
            if (data.success) {
                this.showToast && this.showToast('Batch Copy', 'Copied items successfully', 'success');
                this.fmClearSelection();
                this.fetchFileList(this.currentPath);
            } else {
                this.showToast && this.showToast('Batch Copy Error', data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Batch Copy Error', 'Batch copy failed', 'error');
        }
    },

    async fmMoveSelected() {
        if (this.fmSelectedPaths.length === 0) return;
        try {
            const res = await fetch('/api/files/batch', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ action: 'move', paths: this.fmSelectedPaths, dest_dir: this.currentPath })
            });
            const data = await res.json();
            if (data.success) {
                this.showToast && this.showToast('Batch Move', 'Moved items successfully', 'success');
                this.fmClearSelection();
                this.fetchFileList(this.currentPath);
            } else {
                this.showToast && this.showToast('Batch Move Error', data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Batch Move Error', 'Batch move failed', 'error');
        }
    },

    async fmCompressSelected() {
        if (this.fmSelectedPaths.length === 0) return;
        const zipName = 'archive_' + Date.now() + '.zip';
        const destZip = this.currentPath + (this.currentPath.endsWith('/') ? '' : '/') + zipName;
        try {
            const res = await fetch('/api/files/compress', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ paths: this.fmSelectedPaths, dest_zip: destZip })
            });
            const data = await res.json();
            if (data.success) {
                this.showToast && this.showToast('Archive Created', `Created ${zipName}`, 'success');
                this.fmClearSelection();
                this.fetchFileList(this.currentPath);
            } else {
                this.showToast && this.showToast('Compress Error', data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Compress Error', 'Compression failed', 'error');
        }
    },

    // Permissions Modal & Logic
    openPermModal(file) {
        if (!file || file.is_parent) return;
        this.permTarget = { path: file.path, name: file.name, is_dir: file.is_dir };
        this.permMode = '0755';
        this.permOwner = 'root:root';
        this.permRecursive = false;
        this.showPermModal = true;
    },

    async applyPermissions() {
        if (!this.permTarget.path) return;
        try {
            const res = await fetch('/api/files/permissions', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    path: this.permTarget.path,
                    mode: this.permMode,
                    owner: this.permOwner
                })
            });
            const data = await res.json();
            if (data.success) {
                this.showPermModal = false;
                this.showToast && this.showToast('Permissions Changed', `Updated ${this.permTarget.name}`, 'success');
                this.fetchFileList(this.currentPath);
            } else {
                this.showToast && this.showToast('Permission Error', data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Permission Error', 'Failed to update permissions', 'error');
        }
    },

    // Image Preview & Navigation
    fmIsImage(file) {
        if (!file || file.is_dir || file.is_parent) return false;
        const ext = file.name.split('.').pop().toLowerCase();
        return ['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'svg'].includes(ext);
    },

    openImagePreview(file) {
        if (!this.fmIsImage(file)) return;
        const url = '/api/files/download?path=' + encodeURIComponent(file.path);
        const formatSize = (bytes) => {
            if (bytes === 0) return '0 B';
            const k = 1024;
            const sizes = ['B', 'KB', 'MB', 'GB'];
            const i = Math.floor(Math.log(bytes) / Math.log(k));
            return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
        };
        this.imagePreview = {
            path: file.path,
            name: file.name,
            url: url,
            loading: true,
            error: false,
            size_str: formatSize(file.size || 0),
            width: 0,
            height: 0
        };
        this.showImagePreviewModal = true;
    },

    fmImagePrev() {
        const images = this.fileList.filter(f => this.fmIsImage(f));
        if (images.length === 0) return;
        const currentIdx = images.findIndex(f => f.path === this.imagePreview.path);
        const prevIdx = (currentIdx - 1 + images.length) % images.length;
        this.openImagePreview(images[prevIdx]);
    },

    fmImageNext() {
        const images = this.fileList.filter(f => this.fmIsImage(f));
        if (images.length === 0) return;
        const currentIdx = images.findIndex(f => f.path === this.imagePreview.path);
        const nextIdx = (currentIdx + 1) % images.length;
        this.openImagePreview(images[nextIdx]);
    },

    // ZIP extraction / compression
    async extractZip(file) {
        if (!file || file.is_dir || !file.name.endsWith('.zip')) return;
        try {
            const res = await fetch('/api/files/extract', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ zip_path: file.path, dest_dir: this.currentPath })
            });
            const data = await res.json();
            if (data.success) {
                this.showToast && this.showToast('Extracted Archive', `Extracted ${file.name}`, 'success');
                this.fetchFileList(this.currentPath);
            } else {
                this.showToast && this.showToast('Extract Error', data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Extract Error', 'Extraction request failed', 'error');
        }
    },

    // File Filtering / Search
    async fmFilterFiles() {
        if (this.fmSearchQuery.startsWith('all:')) {
            const query = this.fmSearchQuery.substring(4).trim();
            if (!query) return;
            try {
                const res = await fetch(`/api/files/search?path=${encodeURIComponent(this.currentPath)}&query=${encodeURIComponent(query)}`);
                if (res.ok) {
                    const data = await res.json();
                    this.fileList = data.files || [];
                }
            } catch (e) {}
        }
    },

    fmFilteredList() {
        if (!this.fmSearchQuery || this.fmSearchQuery.startsWith('all:')) {
            return this.fileList;
        }
        const q = this.fmSearchQuery.toLowerCase();
        return this.fileList.filter(f => f.is_parent || f.name.toLowerCase().includes(q));
    },

    // UI Formatting Helpers
    fmFileIcon(file) {
        if (!file) return '📄';
        if (file.is_parent) return '⬆️';
        if (file.is_dir) return '📁';
        const name = (file.name || '').toLowerCase();
        if (name.endsWith('.zip') || name.endsWith('.tar') || name.endsWith('.gz') || name.endsWith('.7z')) return '📦';
        if (this.fmIsImage(file) || name.endsWith('.png') || name.endsWith('.jpg') || name.endsWith('.jpeg') || name.endsWith('.webp') || name.endsWith('.svg') || name.endsWith('.gif')) return '🖼️';
        if (name.endsWith('.txt') || name.endsWith('.md') || name.endsWith('.json') || name.endsWith('.yaml') || name.endsWith('.sh') || name.endsWith('.log') || name.endsWith('.go') || name.endsWith('.prop') || name.endsWith('.conf')) return '📄';
        if (name.endsWith('.apk')) return '📱';
        if (name.endsWith('.mp3') || name.endsWith('.wav') || name.endsWith('.flac') || name.endsWith('.ogg')) return '🎵';
        if (name.endsWith('.mp4') || name.endsWith('.mkv') || name.endsWith('.avi')) return '🎬';
        return '📄';
    },

    fmOctalToSymbolic(mode) {
        if (!mode) return 'rw-r--r--';
        const str = String(mode).trim();
        if (str.length === 10 || str.startsWith('d') || str.startsWith('-')) {
            return str;
        }
        let clean = str;
        if (clean.length === 4 && clean.startsWith('0')) {
            clean = clean.substring(1);
        }
        if (clean.length < 3) {
            clean = clean.padStart(3, '0');
        }
        const mapping = {
            '0': '---',
            '1': '--x',
            '2': '-w-',
            '3': '-wx',
            '4': 'r--',
            '5': 'r-x',
            '6': 'rw-',
            '7': 'rwx'
        };
        const u = mapping[clean[0]] || 'rwx';
        const g = mapping[clean[1]] || 'r-x';
        const o = mapping[clean[2]] || 'r-x';
        return u + g + o;
    },

    fmOctalBit(mode, row, col) {
        if (!mode) return false;
        let str = String(mode).trim();
        if (str.length === 4 && str.startsWith('0')) {
            str = str.substring(1);
        }
        if (str.length < 3) {
            str = str.padStart(3, '0');
        }
        const digitChar = str[row];
        if (!digitChar) return false;
        const num = parseInt(digitChar, 10);
        if (isNaN(num)) return false;

        if (col === 0) return (num & 4) !== 0; // Read bit
        if (col === 1) return (num & 2) !== 0; // Write bit
        if (col === 2) return (num & 1) !== 0; // Execute bit
        return false;
    },

    // Existing Dialog / Editor Methods
    async openEditor(filePath) {
        try {
            const res = await fetch('/api/files/read?path=' + encodeURIComponent(filePath));
            if (res.ok) {
                const data = await res.json();
                this.editingFilePath = data.path;
                this.editorContent = data.content;
                this.showEditorModal = true;
            } else {
                this.showToast && this.showToast('Read Error', 'Cannot read file (may exceed 5MB or be binary)', 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Read Error', 'Failed to retrieve file content.', 'error');
        }
    },

    async saveFileContent() {
        try {
            const res = await fetch('/api/files/save', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path: this.editingFilePath, content: this.editorContent })
            });
            const data = await res.json();
            if (data.success) {
                this.showEditorModal = false;
                this.fetchFileList(this.currentPath);
                this.showToast && this.showToast('File Saved', 'File contents written successfully!', 'success');
            } else {
                this.showToast && this.showToast('Save Error', data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Save Error', 'File writing request failed.', 'error');
        }
    },

    async uploadFile() {
        const input = document.getElementById('upload-input');
        if (!input.files.length) return;

        const formData = new FormData();
        formData.append('path', this.currentPath);
        formData.append('file', input.files[0]);

        try {
            const res = await fetch('/api/files/upload', {
                method: 'POST',
                body: formData
            });
            const data = await res.json();
            if (data.success) {
                this.showUploadModal = false;
                this.fetchFileList(this.currentPath);
                this.showToast && this.showToast('Upload Success', 'File uploaded successfully!', 'success');
            } else {
                this.showToast && this.showToast('Upload Error', 'Upload failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Upload Error', 'File upload request failed.', 'error');
        }
    },

    async handleFileDrop(event) {
        this.isDragging = false;
        const files = event.dataTransfer.files;
        if (!files.length) return;

        const formData = new FormData();
        formData.append('path', this.currentPath);
        formData.append('file', files[0]);

        this.showToast && this.showToast('Uploading Drop', 'Uploading dropped file...', 'info');
        try {
            const res = await fetch('/api/files/upload', {
                method: 'POST',
                body: formData
            });
            const data = await res.json();
            if (data.success) {
                this.fetchFileList(this.currentPath);
                this.showToast && this.showToast('Upload Success', 'Dropped file uploaded successfully!', 'success');
            } else {
                this.showToast && this.showToast('Upload Error', 'Upload failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Upload Error', 'Dropped file upload failed.', 'error');
        }
    },

    openCreateDirModal() {
        this.newDirName = '';
        this.showDirModal = true;
    },

    async createDir() {
        if (!this.newDirName) return;
        const fullPath = this.currentPath + '/' + this.newDirName;
        try {
            const res = await fetch('/api/files/mkdir', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path: fullPath })
            });
            const data = await res.json();
            if (data.success) {
                this.showDirModal = false;
                this.fetchFileList(this.currentPath);
                this.showToast && this.showToast('Folder Created', 'New directory created successfully!', 'success');
            } else {
                this.showToast && this.showToast('Directory Error', 'Create directory failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Directory Error', 'Create directory request failed.', 'error');
        }
    },

    deletePath(targetPath) {
        this.showConfirm(
            'Delete Confirmation',
            `Are you sure you want to delete: ${targetPath}?`,
            async () => {
                try {
                    const res = await fetch('/api/files/delete', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ path: targetPath })
                    });
                    const data = await res.json();
                    if (data.success) {
                        this.fetchFileList(this.currentPath);
                        this.showToast && this.showToast('Delete Success', 'Item deleted successfully!', 'success');
                    } else {
                        this.showToast && this.showToast('Delete Error', 'Delete failed: ' + data.error, 'error');
                    }
                } catch (e) {
                    this.showToast && this.showToast('Delete Error', 'Delete request failed.', 'error');
                }
            }
        );
    },

    openCreateFileModal() {
        this.newFileName = '';
        this.showFileModal = true;
    },

    async createFile() {
        if (!this.newFileName) return;
        const fullPath = this.currentPath + (this.currentPath.endsWith('/') ? '' : '/') + this.newFileName.trim();
        try {
            const res = await fetch('/api/files/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path: fullPath })
            });
            const data = await res.json();
            if (data.success) {
                this.showFileModal = false;
                this.fetchFileList(this.currentPath);
                this.showToast && this.showToast('File Created', 'New empty file created successfully!', 'success');
            } else {
                this.showToast && this.showToast('Create File Error', 'Create file failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Create File Error', 'Create file request failed.', 'error');
        }
    },

    openRenameModal(file) {
        this.renameOldPath = file.path;
        this.renameNewName = file.name;
        this.showRenameModal = true;
    },

    async renamePath() {
        if (!this.renameNewName) return;
        const idx = this.renameOldPath.lastIndexOf('/');
        const parent = idx >= 0 ? this.renameOldPath.substring(0, idx + 1) : '';
        const newPath = parent + this.renameNewName.trim();
        try {
            const res = await fetch('/api/files/rename', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ old_path: this.renameOldPath, new_path: newPath })
            });
            const data = await res.json();
            if (data.success) {
                this.showRenameModal = false;
                this.fetchFileList(this.currentPath);
                this.showToast && this.showToast('Rename Success', 'Item renamed successfully!', 'success');
            } else {
                this.showToast && this.showToast('Rename Error', 'Rename failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast && this.showToast('Rename Error', 'Rename request failed.', 'error');
        }
    },

    initShortcuts() {
        let saved = localStorage.getItem('fileManagerShortcuts');
        if (saved) {
            try {
                this.fileManagerShortcuts = JSON.parse(saved);
            } catch(e) {
                this.fileManagerShortcuts = this.getDefaultShortcuts();
            }
        } else {
            this.fileManagerShortcuts = this.getDefaultShortcuts();
            localStorage.setItem('fileManagerShortcuts', JSON.stringify(this.fileManagerShortcuts));
        }
    },

    getDefaultShortcuts() {
        return [
            { name: "/sdcard", path: "/sdcard" },
            { name: "/data/adb", path: "/data/adb" },
            { name: "/modules", path: "/data/adb/modules" }
        ];
    },

    addShortcut(name, path) {
        if (!name || !path) return;
        this.fileManagerShortcuts.push({ name, path });
        localStorage.setItem('fileManagerShortcuts', JSON.stringify(this.fileManagerShortcuts));
    },

    removeShortcut(index) {
        this.fileManagerShortcuts.splice(index, 1);
        localStorage.setItem('fileManagerShortcuts', JSON.stringify(this.fileManagerShortcuts));
    },

    openAddShortcutModal() {
        this.newShortcutName = '';
        this.newShortcutPath = '';
        this.showAddShortcutModal = true;
    },

    saveShortcut() {
        if (this.newShortcutName && this.newShortcutPath) {
            this.addShortcut(this.newShortcutName.trim(), this.newShortcutPath.trim());
            this.showAddShortcutModal = false;
        }
    }
};
