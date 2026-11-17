// Package platform инкапсулирует взаимодействие с операционной системой для [anyui].
//
// Пакет предоставляет:
//   - интерфейс [Window] и нативную реализацию окна (включая цикл событий [RunEventLoop]);
//   - системные диалоги выбора файлов и папок ([OpenFileDialog], [SaveFileDialog], [SelectFolderDialog]);
//   - иконку в системном трее ([Tray], [NewTray]) и уведомления ([TrayNotification]);
//   - вспомогательные функции ОС: [MessageBox], [OpenURL], [RevealInFileManager],
//     [IsSystemDarkTheme] и защиту от повторного запуска ([AcquireSingleInstance]).
package platform

