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

    async fetchFileList(path) {
        try {
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
            }
        } catch (e) {}
    },

    navigateBreadcrumb(index) {
        const segments = this.currentPathSegments.slice(0, index + 1);
        const targetPath = '/' + segments.join('/');
        this.fetchFileList(targetPath);
    },

    async openEditor(filePath) {
        try {
            const res = await fetch('/api/files/read?path=' + encodeURIComponent(filePath));
            if (res.ok) {
                const data = await res.json();
                this.editingFilePath = data.path;
                this.editorContent = data.content;
                this.showEditorModal = true;
            } else {
                this.showToast('Read Error', 'Cannot read file (may exceed 5MB or be binary)', 'error');
            }
        } catch (e) {
            this.showToast('Read Error', 'Failed to retrieve file content.', 'error');
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
                this.showToast('File Saved', 'File contents written successfully!', 'success');
            } else {
                this.showToast('Save Error', data.error, 'error');
            }
        } catch (e) {
            this.showToast('Save Error', 'File writing request failed.', 'error');
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
                this.showToast('Upload Success', 'File uploaded successfully!', 'success');
            } else {
                this.showToast('Upload Error', 'Upload failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast('Upload Error', 'File upload request failed.', 'error');
        }
    },

    async handleFileDrop(event) {
        this.isDragging = false;
        const files = event.dataTransfer.files;
        if (!files.length) return;

        const formData = new FormData();
        formData.append('path', this.currentPath);
        formData.append('file', files[0]);

        this.showToast('Uploading Drop', 'Uploading dropped file...', 'info');
        try {
            const res = await fetch('/api/files/upload', {
                method: 'POST',
                body: formData
            });
            const data = await res.json();
            if (data.success) {
                this.fetchFileList(this.currentPath);
                this.showToast('Upload Success', 'Dropped file uploaded successfully!', 'success');
            } else {
                this.showToast('Upload Error', 'Upload failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast('Upload Error', 'Dropped file upload failed.', 'error');
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
                this.showToast('Folder Created', 'New directory created successfully!', 'success');
            } else {
                this.showToast('Directory Error', 'Create directory failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast('Directory Error', 'Create directory request failed.', 'error');
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
                        this.showToast('Delete Success', 'Item deleted successfully!', 'success');
                    } else {
                        this.showToast('Delete Error', 'Delete failed: ' + data.error, 'error');
                    }
                } catch (e) {
                    this.showToast('Delete Error', 'Delete request failed.', 'error');
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
                this.showToast('File Created', 'New empty file created successfully!', 'success');
            } else {
                this.showToast('Create File Error', 'Create file failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast('Create File Error', 'Create file request failed.', 'error');
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
                this.showToast('Rename Success', 'Item renamed successfully!', 'success');
            } else {
                this.showToast('Rename Error', 'Rename failed: ' + data.error, 'error');
            }
        } catch (e) {
            this.showToast('Rename Error', 'Rename request failed.', 'error');
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
