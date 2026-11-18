//go:build darwin && cgo

#if defined(__APPLE__) && defined(__MACH__)

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import <objc/runtime.h>
#include <math.h>
#include "darwin_bridge.h"

// Обратные вызовы в Go (экспортируются через //export в darwin_wnd.go).
extern int goDarwinOnClose(uint64_t winID);
extern void goDarwinOnDestroy(uint64_t winID);
extern void goDarwinOnResize(uint64_t winID, int width, int height);
extern void goDarwinOnDPIChange(uint64_t winID, float scale);
extern void goDarwinOnPaint(uint64_t winID);
extern void goDarwinOnWakeUp(uint64_t winID);

static char kAnyUIDelegateKey;

static void runOnMainSync(dispatch_block_t block) {
    if ([NSThread isMainThread]) {
        block();
    } else {
        dispatch_sync(dispatch_get_main_queue(), block);
    }
}

static void runOnMainAsync(dispatch_block_t block) {
    if ([NSThread isMainThread]) {
        block();
    } else {
        dispatch_async(dispatch_get_main_queue(), block);
    }
}

@interface AnyUIView : NSView
@property (nonatomic, assign) uint64_t winID;
@end

@implementation AnyUIView
- (BOOL)wantsUpdateLayer {
    return NO;
}
- (BOOL)acceptsFirstResponder {
    return YES;
}
@end

@interface AnyUIWindowDelegate : NSObject <NSWindowDelegate>
@property (nonatomic, assign) uint64_t winID;
@property (nonatomic, weak) NSWindow *window;
@property (nonatomic, assign) NSRect savedFrame;
@property (nonatomic, assign) BOOL isMaximizedState;
@end

@implementation AnyUIWindowDelegate

- (BOOL)windowShouldClose:(NSWindow *)sender {
    (void)sender;
    return goDarwinOnClose(self.winID) != 0;
}

- (void)windowWillClose:(NSNotification *)notification {
    (void)notification;
    goDarwinOnDestroy(self.winID);
}

- (void)windowDidResize:(NSNotification *)notification {
    (void)notification;
    NSWindow *win = self.window;
    if (!win || !win.contentView) {
        return;
    }
    NSRect bounds = win.contentView.bounds;
    CGFloat scale = win.backingScaleFactor > 0 ? win.backingScaleFactor : 1.0;
    int pxW = (int)llround(bounds.size.width * scale);
    int pxH = (int)llround(bounds.size.height * scale);
    if (pxW < 1) pxW = 1;
    if (pxH < 1) pxH = 1;
    goDarwinOnResize(self.winID, pxW, pxH);
    goDarwinOnPaint(self.winID);
}

- (void)windowDidChangeBackingProperties:(NSNotification *)notification {
    (void)notification;
    NSWindow *win = self.window;
    if (!win || !win.contentView) {
        return;
    }
    CGFloat scale = win.backingScaleFactor > 0 ? win.backingScaleFactor : 1.0;
    goDarwinOnDPIChange(self.winID, (float)scale);

    NSRect bounds = win.contentView.bounds;
    int pxW = (int)llround(bounds.size.width * scale);
    int pxH = (int)llround(bounds.size.height * scale);
    if (pxW < 1) pxW = 1;
    if (pxH < 1) pxH = 1;
    goDarwinOnResize(self.winID, pxW, pxH);
    goDarwinOnPaint(self.winID);
}

@end

static AnyUIWindowDelegate* getDelegateForWindow(NSWindow *win) {
    if (!win) return nil;
    return (AnyUIWindowDelegate *)objc_getAssociatedObject(win, &kAnyUIDelegateKey);
}

void anyui_darwin_init_app(void) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    }
}

void anyui_darwin_run_app(void) {
    @autoreleasepool {
        [NSApp activateIgnoringOtherApps:YES];
        [NSApp run];
    }
}

void anyui_darwin_stop_app(void) {
    runOnMainAsync(^{
        @autoreleasepool {
            [NSApp stop:nil];
            // Отправляем фиктивное событие, чтобы [NSApp run] вышел немедленно.
            NSEvent *ev = [NSEvent otherEventWithType:NSEventTypeApplicationDefined
                                             location:NSZeroPoint
                                        modifierFlags:0
                                            timestamp:0
                                         windowNumber:0
                                              context:nil
                                              subtype:0
                                                data1:0
                                                data2:0];
            if (ev) {
                [NSApp postEvent:ev atStart:YES];
            }
        }
    });
}

uintptr_t anyui_darwin_create_window(
    uint64_t winID,
    const char* title,
    int width,
    int height,
    int minWidth,
    int minHeight,
    int darkTitle,
    int frameless
) {
    __block uintptr_t result = 0;
    NSString *titleStr = title ? [NSString stringWithUTF8String:title] : @"";

    runOnMainSync(^{
        @autoreleasepool {
            NSWindowStyleMask style = NSWindowStyleMaskTitled |
                                      NSWindowStyleMaskClosable |
                                      NSWindowStyleMaskMiniaturizable |
                                      NSWindowStyleMaskResizable;
            if (frameless) {
                style = NSWindowStyleMaskBorderless | NSWindowStyleMaskResizable | NSWindowStyleMaskMiniaturizable;
            }

            NSRect rect = NSMakeRect(0, 0, width > 0 ? width : 800, height > 0 ? height : 600);
            NSWindow *win = [[NSWindow alloc] initWithContentRect:rect
                                                        styleMask:style
                                                          backing:NSBackingStoreBuffered
                                                            defer:NO];
            if (!win) {
                return;
            }

            [win setTitle:titleStr ? titleStr : @""];
            [win center];
            [win setReleasedWhenClosed:NO];

            if (minWidth > 0 || minHeight > 0) {
                [win setContentMinSize:NSMakeSize(minWidth > 0 ? minWidth : 100,
                                                  minHeight > 0 ? minHeight : 100)];
            }

            if (darkTitle) {
                win.appearance = [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua];
            }

            AnyUIView *view = [[AnyUIView alloc] initWithFrame:rect];
            view.winID = winID;
            view.wantsLayer = YES;
            view.layerContentsRedrawPolicy = NSViewLayerContentsRedrawDuringViewResize;
            [win setContentView:view];

            AnyUIWindowDelegate *delegate = [[AnyUIWindowDelegate alloc] init];
            delegate.winID = winID;
            delegate.window = win;
            objc_setAssociatedObject(win, &kAnyUIDelegateKey, delegate, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
            [win setDelegate:delegate];

            result = (uintptr_t)CFBridgingRetain(win);
        }
    });

    return result;
}

void anyui_darwin_destroy_window(uintptr_t handle) {
    if (!handle) return;
    runOnMainAsync(^{
        @autoreleasepool {
            NSWindow *win = (NSWindow *)CFBridgingRelease((CFTypeRef)handle);
            [win setDelegate:nil];
            objc_setAssociatedObject(win, &kAnyUIDelegateKey, nil, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
        }
    });
}

void anyui_darwin_show_window(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            [win makeKeyAndOrderFront:nil];
            [NSApp activateIgnoringOtherApps:YES];
        }
    });
}

void anyui_darwin_hide_window(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            [win orderOut:nil];
        }
    });
}

void anyui_darwin_close_window(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            [win performClose:nil];
        }
    });
}

void anyui_darwin_set_title(uintptr_t handle, const char* title) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    NSString *titleStr = title ? [NSString stringWithUTF8String:title] : @"";
    runOnMainAsync(^{
        @autoreleasepool {
            [win setTitle:titleStr ? titleStr : @""];
        }
    });
}

void anyui_darwin_present(uintptr_t handle, const void* bgraPixels, int width, int height) {
    if (!handle || !bgraPixels || width <= 0 || height <= 0) return;

    size_t byteCount = (size_t)width * (size_t)height * 4;
    CFDataRef data = CFDataCreate(kCFAllocatorDefault, (const UInt8 *)bgraPixels, (CFIndex)byteCount);
    if (!data) return;

    CGDataProviderRef provider = CGDataProviderCreateWithCFData(data);
    CFRelease(data);
    if (!provider) return;

    CGColorSpaceRef colorSpace = CGColorSpaceCreateDeviceRGB();
    CGBitmapInfo bitmapInfo = kCGBitmapByteOrder32Little | kCGImageAlphaPremultipliedFirst;
    CGImageRef image = CGImageCreate(
        (size_t)width,
        (size_t)height,
        8,
        32,
        (size_t)width * 4,
        colorSpace,
        bitmapInfo,
        provider,
        NULL,
        false,
        kCGRenderingIntentDefault
    );
    CGColorSpaceRelease(colorSpace);
    CGDataProviderRelease(provider);
    if (!image) return;

    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            NSView *view = win.contentView;
            if (view && view.layer) {
                CGFloat scale = win.backingScaleFactor > 0 ? win.backingScaleFactor : 1.0;
                [CATransaction begin];
                [CATransaction setDisableActions:YES];
                view.layer.contentsScale = scale;
                view.layer.contents = (__bridge id)image;
                [CATransaction commit];
            }
            CGImageRelease(image);
        }
    });
}

void anyui_darwin_invalidate(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            AnyUIWindowDelegate *del = getDelegateForWindow(win);
            if (del) {
                goDarwinOnPaint(del.winID);
            }
        }
    });
}

void anyui_darwin_wakeup(uint64_t winID) {
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            goDarwinOnWakeUp(winID);
        }
    });
}

void anyui_darwin_get_size(uintptr_t handle, int* outW, int* outH, float* outScale) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainSync(^{
        @autoreleasepool {
            CGFloat scale = win.backingScaleFactor > 0 ? win.backingScaleFactor : 1.0;
            NSRect bounds = win.contentView ? win.contentView.bounds : NSZeroRect;
            if (outW) *outW = (int)llround(bounds.size.width * scale);
            if (outH) *outH = (int)llround(bounds.size.height * scale);
            if (outScale) *outScale = (float)scale;
        }
    });
}

void anyui_darwin_set_size(uintptr_t handle, int width, int height) {
    if (!handle || width <= 0 || height <= 0) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            [win setContentSize:NSMakeSize(width, height)];
        }
    });
}

void anyui_darwin_get_position(uintptr_t handle, int* outX, int* outY) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainSync(^{
        @autoreleasepool {
            NSRect frame = win.frame;
            NSScreen *screen = win.screen ? win.screen : [NSScreen mainScreen];
            CGFloat screenH = screen ? screen.frame.size.height : frame.size.height;
            CGFloat scale = win.backingScaleFactor > 0 ? win.backingScaleFactor : 1.0;
            if (outX) *outX = (int)llround(frame.origin.x * scale);
            if (outY) *outY = (int)llround((screenH - (frame.origin.y + frame.size.height)) * scale);
        }
    });
}

void anyui_darwin_set_position(uintptr_t handle, int x, int y) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            NSRect frame = win.frame;
            NSScreen *screen = win.screen ? win.screen : [NSScreen mainScreen];
            CGFloat screenH = screen ? screen.frame.size.height : frame.size.height;
            CGFloat scale = win.backingScaleFactor > 0 ? win.backingScaleFactor : 1.0;
            CGFloat ptsX = (CGFloat)x / scale;
            CGFloat ptsTopY = (CGFloat)y / scale;
            CGFloat ptsBottomY = screenH - ptsTopY - frame.size.height;
            [win setFrameOrigin:NSMakePoint(ptsX, ptsBottomY)];
        }
    });
}

void anyui_darwin_minimize(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            [win miniaturize:nil];
        }
    });
}

void anyui_darwin_maximize(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            AnyUIWindowDelegate *del = getDelegateForWindow(win);
            if (del && !del.isMaximizedState) {
                del.savedFrame = win.frame;
                del.isMaximizedState = YES;
            }
            NSScreen *screen = win.screen ? win.screen : [NSScreen mainScreen];
            if (screen) {
                [win setFrame:screen.visibleFrame display:YES];
            }
        }
    });
}

void anyui_darwin_restore(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            if ([win isMiniaturized]) {
                [win deminiaturize:nil];
            }
            AnyUIWindowDelegate *del = getDelegateForWindow(win);
            if (del && del.isMaximizedState) {
                del.isMaximizedState = NO;
                if (!NSIsEmptyRect(del.savedFrame)) {
                    [win setFrame:del.savedFrame display:YES];
                }
            }
        }
    });
}

int anyui_darwin_is_maximized(uintptr_t handle) {
    if (!handle) return 0;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    __block int res = 0;
    runOnMainSync(^{
        @autoreleasepool {
            AnyUIWindowDelegate *del = getDelegateForWindow(win);
            res = (del && del.isMaximizedState) ? 1 : 0;
        }
    });
    return res;
}

int anyui_darwin_is_minimized(uintptr_t handle) {
    if (!handle) return 0;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    __block int res = 0;
    runOnMainSync(^{
        @autoreleasepool {
            res = [win isMiniaturized] ? 1 : 0;
        }
    });
    return res;
}

void anyui_darwin_set_fullscreen(uintptr_t handle, int fullscreen) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            BOOL isFS = (win.styleMask & NSWindowStyleMaskFullScreen) != 0;
            if ((fullscreen != 0) != isFS) {
                [win toggleFullScreen:nil];
            }
        }
    });
}

int anyui_darwin_is_fullscreen(uintptr_t handle) {
    if (!handle) return 0;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    __block int res = 0;
    runOnMainSync(^{
        @autoreleasepool {
            res = (win.styleMask & NSWindowStyleMaskFullScreen) != 0 ? 1 : 0;
        }
    });
    return res;
}

void anyui_darwin_bring_to_front(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            [win makeKeyAndOrderFront:nil];
            [NSApp activateIgnoringOtherApps:YES];
        }
    });
}

void anyui_darwin_set_dark_title(uintptr_t handle, int dark) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            if (dark) {
                win.appearance = [NSAppearance appearanceNamed:NSAppearanceNameDarkAqua];
            } else {
                win.appearance = [NSAppearance appearanceNamed:NSAppearanceNameAqua];
            }
        }
    });
}

int anyui_darwin_double_click_ms(void) {
    NSTimeInterval interval = [NSEvent doubleClickInterval];
    if (interval <= 0) {
        return 500;
    }
    return (int)llround(interval * 1000.0);
}

#endif // __APPLE__
