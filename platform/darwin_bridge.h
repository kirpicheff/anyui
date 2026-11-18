#ifndef ANYUI_DARWIN_BRIDGE_H
#define ANYUI_DARWIN_BRIDGE_H

#include <stdint.h>

// Инициализация NSApplication и управление главным циклом событий.
void anyui_darwin_init_app(void);
void anyui_darwin_run_app(void);
void anyui_darwin_stop_app(void);

// Создание и управление NSWindow.
uintptr_t anyui_darwin_create_window(
    uint64_t winID,
    const char* title,
    int width,
    int height,
    int minWidth,
    int minHeight,
    int darkTitle,
    int frameless
);
void anyui_darwin_destroy_window(uintptr_t handle);
void anyui_darwin_show_window(uintptr_t handle);
void anyui_darwin_hide_window(uintptr_t handle);
void anyui_darwin_close_window(uintptr_t handle);
void anyui_darwin_set_title(uintptr_t handle, const char* title);

// Отрисовка BGRA-буфера через CALayer.contents и управление перерисовкой.
void anyui_darwin_present(uintptr_t handle, const void* bgraPixels, int width, int height);
void anyui_darwin_invalidate(uintptr_t handle);
void anyui_darwin_wakeup(uint64_t winID);

// Геометрия, масштабирование и состояние окна.
void anyui_darwin_get_size(uintptr_t handle, int* outW, int* outH, float* outScale);
void anyui_darwin_set_size(uintptr_t handle, int width, int height);
void anyui_darwin_get_position(uintptr_t handle, int* outX, int* outY);
void anyui_darwin_set_position(uintptr_t handle, int x, int y);
void anyui_darwin_minimize(uintptr_t handle);
void anyui_darwin_maximize(uintptr_t handle);
void anyui_darwin_restore(uintptr_t handle);
int anyui_darwin_is_maximized(uintptr_t handle);
int anyui_darwin_is_minimized(uintptr_t handle);
void anyui_darwin_set_fullscreen(uintptr_t handle, int fullscreen);
int anyui_darwin_is_fullscreen(uintptr_t handle);
void anyui_darwin_bring_to_front(uintptr_t handle);
void anyui_darwin_set_dark_title(uintptr_t handle, int dark);
int anyui_darwin_double_click_ms(void);
void anyui_darwin_set_cursor(uintptr_t handle, int cursorType);
void anyui_darwin_set_capture(uintptr_t handle);
void anyui_darwin_release_capture(uintptr_t handle);

#endif // ANYUI_DARWIN_BRIDGE_H

