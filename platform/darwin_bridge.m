//go:build darwin && cgo

#if defined(__APPLE__) && defined(__MACH__)

#import <Cocoa/Cocoa.h>
#import <QuartzCore/QuartzCore.h>
#import <objc/runtime.h>
#include <math.h>
#include <stdatomic.h>
#include <stdio.h>
#include "darwin_bridge.h"

// Обратные вызовы в Go (экспортируются через //export в darwin_wnd.go).
extern int goDarwinOnClose(uint64_t winID);
extern void goDarwinOnDestroy(uint64_t winID);
extern void goDarwinOnResize(uint64_t winID, int width, int height);
extern void goDarwinOnDPIChange(uint64_t winID, float scale);
extern void goDarwinOnPaint(uint64_t winID);
extern void goDarwinOnWakeUp(uint64_t winID);
extern void goDarwinOnMouse(uint64_t winID, int eventKind, double x, double y, double viewH, double scale, int button, uint64_t mods);
extern void goDarwinOnWheel(uint64_t winID, double x, double y, double viewH, double scale, double deltaX, double deltaY, int precise, uint64_t mods);
extern void goDarwinOnKey(uint64_t winID, int isDown, uint16_t keyCode, uint64_t mods, const char* charsUTF8);
extern void goDarwinOnFlagsChanged(uint64_t winID, uint64_t mods);
extern void goDarwinOnFocusLost(uint64_t winID);
extern int goDarwinHitTest(uint64_t winID, double x, double y, double viewH, double scale);
extern void goDarwinOnDropFiles(uint64_t winID, const char* pathsNewlineSep);

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

@interface AnyUIView : NSView <NSDraggingDestination>
@property (nonatomic, assign) uint64_t winID;
@property (nonatomic, assign) BOOL frameless;
@property (nonatomic, assign) BOOL acceptDropFiles;
@property (nonatomic, assign) CGImageRef lastImage;
@property (nonatomic, strong) NSTrackingArea *trackingArea;
@property (nonatomic, strong) NSCursor *currentCursor;
@end

@implementation AnyUIView
- (void)dealloc {
    if (_lastImage) {
        CGImageRelease(_lastImage);
        _lastImage = NULL;
    }
}
- (BOOL)wantsUpdateLayer {
    return YES;
}
- (void)updateLayer {
    CALayer *layer = self.layer;
    if (!layer) return;
    CGFloat scale = (self.window && self.window.backingScaleFactor > 0) ? self.window.backingScaleFactor : 1.0;
    [CATransaction begin];
    [CATransaction setDisableActions:YES];
    layer.contentsScale = scale;
    if (_lastImage) {
        layer.contents = (__bridge id)_lastImage;
    }
    [CATransaction commit];
}
- (void)setPresentedImage:(CGImageRef)img {
    if (_lastImage != img) {
        if (_lastImage) {
            CGImageRelease(_lastImage);
        }
        _lastImage = img ? CGImageRetain(img) : NULL;
    }
    if (self.layer) {
        [self updateLayer];
    } else {
        [self setNeedsDisplay:YES];
    }
}
- (BOOL)acceptsFirstResponder {
    return YES;
}
- (BOOL)mouseDownCanMoveWindow {
    return NO;
}

- (void)updateTrackingAreas {
    [super updateTrackingAreas];
    if (self.trackingArea) {
        [self removeTrackingArea:self.trackingArea];
        self.trackingArea = nil;
    }
    NSTrackingAreaOptions opts = NSTrackingMouseMoved |
                                 NSTrackingActiveInKeyWindow |
                                 NSTrackingInVisibleRect;
    self.trackingArea = [[NSTrackingArea alloc] initWithRect:self.bounds
                                                     options:opts
                                                       owner:self
                                                    userInfo:nil];
    [self addTrackingArea:self.trackingArea];
}

- (void)resetCursorRects {
    if (self.currentCursor) {
        [self addCursorRect:self.bounds cursor:self.currentCursor];
    } else {
        [super resetCursorRects];
    }
}

- (void)cursorUpdate:(NSEvent *)event {
    if (self.currentCursor) {
        [self.currentCursor set];
    } else {
        [super cursorUpdate:event];
    }
}

- (void)sendMouseEvent:(NSEvent *)event kind:(int)kind button:(int)button {
    NSPoint pt = [self convertPoint:[event locationInWindow] fromView:nil];
    CGFloat viewH = self.bounds.size.height;
    CGFloat scale = (self.window && self.window.backingScaleFactor > 0) ? self.window.backingScaleFactor : 1.0;
    goDarwinOnMouse(self.winID, kind, pt.x, pt.y, viewH, scale, button, (uint64_t)event.modifierFlags);
}

- (void)mouseDown:(NSEvent *)event {
    if (self.frameless && self.window) {
        NSPoint pt = [self convertPoint:[event locationInWindow] fromView:nil];
        CGFloat viewH = self.bounds.size.height;
        CGFloat scale = self.window.backingScaleFactor > 0 ? self.window.backingScaleFactor : 1.0;
        if (goDarwinHitTest(self.winID, pt.x, pt.y, viewH, scale) == 1) {
            [self.window performWindowDragWithEvent:event];
            return;
        }
    }
    [self sendMouseEvent:event kind:2 button:0];
}

- (void)mouseUp:(NSEvent *)event {
    [self sendMouseEvent:event kind:3 button:0];
}

- (void)rightMouseDown:(NSEvent *)event {
    [self sendMouseEvent:event kind:2 button:1];
}

- (void)rightMouseUp:(NSEvent *)event {
    [self sendMouseEvent:event kind:3 button:1];
}

- (void)otherMouseDown:(NSEvent *)event {
    if (event.buttonNumber == 2) {
        [self sendMouseEvent:event kind:2 button:2];
    }
}

- (void)otherMouseUp:(NSEvent *)event {
    if (event.buttonNumber == 2) {
        [self sendMouseEvent:event kind:3 button:2];
    }
}

- (void)mouseMoved:(NSEvent *)event {
    [self sendMouseEvent:event kind:1 button:0];
}

- (void)mouseDragged:(NSEvent *)event {
    [self sendMouseEvent:event kind:1 button:0];
}

- (void)rightMouseDragged:(NSEvent *)event {
    [self sendMouseEvent:event kind:1 button:1];
}

- (void)otherMouseDragged:(NSEvent *)event {
    [self sendMouseEvent:event kind:1 button:2];
}

- (void)scrollWheel:(NSEvent *)event {
    NSPoint pt = [self convertPoint:[event locationInWindow] fromView:nil];
    CGFloat viewH = self.bounds.size.height;
    CGFloat scale = (self.window && self.window.backingScaleFactor > 0) ? self.window.backingScaleFactor : 1.0;
    int precise = [event hasPreciseScrollingDeltas] ? 1 : 0;
    goDarwinOnWheel(self.winID, pt.x, pt.y, viewH, scale, event.scrollingDeltaX, event.scrollingDeltaY, precise, (uint64_t)event.modifierFlags);
}

- (void)keyDown:(NSEvent *)event {
    const char *utf8 = event.characters ? [event.characters UTF8String] : NULL;
    goDarwinOnKey(self.winID, 1, (uint16_t)event.keyCode, (uint64_t)event.modifierFlags, utf8);
}

- (void)keyUp:(NSEvent *)event {
    goDarwinOnKey(self.winID, 0, (uint16_t)event.keyCode, (uint64_t)event.modifierFlags, NULL);
}

- (void)flagsChanged:(NSEvent *)event {
    goDarwinOnFlagsChanged(self.winID, (uint64_t)event.modifierFlags);
}

- (NSDragOperation)draggingEntered:(id<NSDraggingInfo>)sender {
    if (!self.acceptDropFiles) {
        return NSDragOperationNone;
    }
    NSPasteboard *pb = [sender draggingPasteboard];
    if ([pb canReadObjectForClasses:@[[NSURL class]]
                            options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}]) {
        return NSDragOperationCopy;
    }
    return NSDragOperationNone;
}

- (BOOL)performDragOperation:(id<NSDraggingInfo>)sender {
    if (!self.acceptDropFiles) {
        return NO;
    }
    NSPasteboard *pb = [sender draggingPasteboard];
    NSArray<NSURL *> *urls = [pb readObjectsForClasses:@[[NSURL class]]
                                               options:@{NSPasteboardURLReadingFileURLsOnlyKey: @YES}];
    if (!urls || urls.count == 0) {
        return NO;
    }
    NSMutableArray<NSString *> *paths = [NSMutableArray arrayWithCapacity:urls.count];
    for (NSURL *u in urls) {
        if (u.path && u.path.length > 0) {
            [paths addObject:u.path];
        }
    }
    if (paths.count == 0) {
        return NO;
    }
    NSString *joined = [paths componentsJoinedByString:@"\n"];
    goDarwinOnDropFiles(self.winID, [joined UTF8String]);
    return YES;
}
@end

@interface AnyUIWindowDelegate : NSObject <NSWindowDelegate> {
@public
    atomic_bool _pendingPaint;
}
@property (nonatomic, assign) uint64_t winID;
@property (nonatomic, weak) NSWindow *window;
@property (nonatomic, assign) NSRect savedFrame;
@property (nonatomic, assign) BOOL isMaximizedState;
@property (nonatomic, assign) BOOL pendingHideOnExitFullScreen;
@property (nonatomic, strong) id localMonitor;
@property (nonatomic, strong) id globalMonitor;
@end

@implementation AnyUIWindowDelegate

- (void)releaseCaptureMonitors {
    if (self.localMonitor) {
        [NSEvent removeMonitor:self.localMonitor];
        self.localMonitor = nil;
    }
    if (self.globalMonitor) {
        [NSEvent removeMonitor:self.globalMonitor];
        self.globalMonitor = nil;
    }
}

- (BOOL)windowShouldClose:(NSWindow *)sender {
    (void)sender;
    return goDarwinOnClose(self.winID) != 0;
}

- (void)windowWillClose:(NSNotification *)notification {
    (void)notification;
    [self releaseCaptureMonitors];
    goDarwinOnDestroy(self.winID);
}

- (void)windowDidResignKey:(NSNotification *)notification {
    (void)notification;
    [self releaseCaptureMonitors];
    goDarwinOnFocusLost(self.winID);
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

- (void)windowDidExitFullScreen:(NSNotification *)notification {
    (void)notification;
    if (self.pendingHideOnExitFullScreen) {
        self.pendingHideOnExitFullScreen = NO;
        NSWindow *win = self.window;
        if (win) {
            [win orderOut:nil];
        }
    }
}

@end

// Делегат приложения для обработки клика по иконке в Dock.
@interface AnyUIAppDelegate : NSObject <NSApplicationDelegate>
@end

@implementation AnyUIAppDelegate

- (BOOL)applicationShouldHandleReopen:(NSApplication *)sender hasVisibleWindows:(BOOL)flag {
    if (!flag) {
        for (NSWindow *win in [sender windows]) {
            if ([win canBecomeKeyWindow] && ![win isKindOfClass:[NSPanel class]]) {
                [win makeKeyAndOrderFront:nil];
                [sender activateIgnoringOtherApps:YES];
                return YES;
            }
        }
    }
    return YES;
}

@end

static AnyUIAppDelegate *g_anyui_app_delegate = nil;

static AnyUIWindowDelegate* getDelegateForWindow(NSWindow *win) {
    if (!win) return nil;
    return (AnyUIWindowDelegate *)objc_getAssociatedObject(win, &kAnyUIDelegateKey);
}

void anyui_darwin_init_app(void) {
    @autoreleasepool {
        BOOL isMain = [NSThread isMainThread];
        fprintf(stderr, "[anyui-darwin] anyui_darwin_init_app: isMainThread=%d\n", isMain ? 1 : 0);
        if (!isMain) {
            fprintf(stderr, "[anyui-darwin] ОШИБКА: anyui_darwin_init_app вызван НЕ на главном потоке ОС!\n");
        }
        fflush(stderr);
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        if (!g_anyui_app_delegate) {
            g_anyui_app_delegate = [[AnyUIAppDelegate alloc] init];
            [NSApp setDelegate:g_anyui_app_delegate];
        }
    }
}

void anyui_darwin_run_app(void) {
    @autoreleasepool {
        BOOL isMain = [NSThread isMainThread];
        fprintf(stderr, "[anyui-darwin] anyui_darwin_run_app: isMainThread=%d\n", isMain ? 1 : 0);
        fflush(stderr);
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
    int frameless,
    int acceptDropFiles
) {
    __block uintptr_t result = 0;
    NSString *titleStr = title ? [NSString stringWithUTF8String:title] : @"";
    fprintf(stderr, "[anyui-darwin] anyui_darwin_create_window(id=%llu): isMainThread=%d\n",
            (unsigned long long)winID, [NSThread isMainThread] ? 1 : 0);
    fflush(stderr);

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
            view.frameless = (frameless != 0);
            view.acceptDropFiles = (acceptDropFiles != 0);
            view.wantsLayer = YES;
            view.layerContentsRedrawPolicy = NSViewLayerContentsRedrawOnSetNeedsDisplay;
            if (acceptDropFiles != 0) {
                [view registerForDraggedTypes:@[NSPasteboardTypeFileURL]];
            }
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
    fprintf(stderr, "[anyui-darwin] anyui_darwin_show_window: isMainThread=%d\n",
            [NSThread isMainThread] ? 1 : 0);
    fflush(stderr);
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            AnyUIWindowDelegate *del = getDelegateForWindow(win);
            if (del) {
                del.pendingHideOnExitFullScreen = NO;
            }
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
            if ((win.styleMask & NSWindowStyleMaskFullScreen) != 0) {
                AnyUIWindowDelegate *del = getDelegateForWindow(win);
                if (del) {
                    del.pendingHideOnExitFullScreen = YES;
                }
                [win toggleFullScreen:nil];
            } else {
                [win orderOut:nil];
            }
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
            NSView *contentView = win.contentView;
            if ([contentView isKindOfClass:[AnyUIView class]]) {
                [(AnyUIView *)contentView setPresentedImage:image];
            } else if (contentView && contentView.layer) {
                CGFloat scale = win.backingScaleFactor > 0 ? win.backingScaleFactor : 1.0;
                [CATransaction begin];
                [CATransaction setDisableActions:YES];
                contentView.layer.contentsScale = scale;
                contentView.layer.contents = (__bridge id)image;
                [CATransaction commit];
            }
            CGImageRelease(image);
        }
    });
}

void anyui_darwin_invalidate(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    AnyUIWindowDelegate *del = getDelegateForWindow(win);
    if (!del) return;

    // Всегда планируем перерисовку асинхронно на следующей итерации RunLoop и схлопываем дубли.
    if (atomic_exchange_explicit(&del->_pendingPaint, true, memory_order_acq_rel)) {
        return;
    }
    dispatch_async(dispatch_get_main_queue(), ^{
        @autoreleasepool {
            atomic_store_explicit(&del->_pendingPaint, false, memory_order_release);
            goDarwinOnPaint(del.winID);
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

void anyui_darwin_set_always_on_top(uintptr_t handle, int onTop) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            [win setLevel:(onTop ? NSFloatingWindowLevel : NSNormalWindowLevel)];
        }
    });
}

int anyui_darwin_is_always_on_top(uintptr_t handle) {
    if (!handle) return 0;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    __block int res = 0;
    runOnMainSync(^{
        @autoreleasepool {
            res = (win.level >= NSFloatingWindowLevel) ? 1 : 0;
        }
    });
    return res;
}

void anyui_darwin_bring_to_front(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            AnyUIWindowDelegate *del = getDelegateForWindow(win);
            if (del) {
                del.pendingHideOnExitFullScreen = NO;
            }
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

static NSCursor* resolveDarwinCursor(int cursorType) {
    switch (cursorType) {
        case 1: // CursorHand
            return [NSCursor pointingHandCursor];
        case 2: // CursorIBeam
            return [NSCursor IBeamCursor];
        case 3: // CursorResizeH
            return [NSCursor resizeLeftRightCursor];
        case 4: // CursorResizeV
            return [NSCursor resizeUpDownCursor];
        case 5: { // CursorResizeNWSE
            SEL sel = NSSelectorFromString(@"_windowResizeNorthWestSouthEastCursor");
            if ([NSCursor respondsToSelector:sel]) {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Warc-performSelector-leaks"
                NSCursor *c = [NSCursor performSelector:sel];
#pragma clang diagnostic pop
                if (c) return c;
            }
            return [NSCursor crosshairCursor];
        }
        case 6: { // CursorResizeNESW
            SEL sel = NSSelectorFromString(@"_windowResizeNorthEastSouthWestCursor");
            if ([NSCursor respondsToSelector:sel]) {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Warc-performSelector-leaks"
                NSCursor *c = [NSCursor performSelector:sel];
#pragma clang diagnostic pop
                if (c) return c;
            }
            return [NSCursor crosshairCursor];
        }
        case 7: // CursorNotAllowed
            return [NSCursor operationNotAllowedCursor];
        case 8: // CursorCrosshair
            return [NSCursor crosshairCursor];
        case 9: // CursorMove
            return [NSCursor openHandCursor];
        case 10: { // CursorNone
            static NSCursor *emptyCursor = nil;
            if (!emptyCursor) {
                NSImage *img = [[NSImage alloc] initWithSize:NSMakeSize(1, 1)];
                emptyCursor = [[NSCursor alloc] initWithImage:img hotSpot:NSZeroPoint];
            }
            return emptyCursor;
        }
        default: // CursorArrow
            return [NSCursor arrowCursor];
    }
}

void anyui_darwin_set_cursor(uintptr_t handle, int cursorType) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            NSCursor *cursor = resolveDarwinCursor(cursorType);
            if ([win.contentView isKindOfClass:[AnyUIView class]]) {
                AnyUIView *view = (AnyUIView *)win.contentView;
                view.currentCursor = cursor;
                [win invalidateCursorRectsForView:view];
            }
            [cursor set];
        }
    });
}

static void dispatchCapturedMouseEvent(NSWindow *win, AnyUIWindowDelegate *del, NSEvent *event) {
    if (!win || !del || !win.contentView) return;
    NSPoint screenPt = [NSEvent mouseLocation];
    NSRect screenRect = NSMakeRect(screenPt.x, screenPt.y, 0, 0);
    NSRect winRect = [win convertRectFromScreen:screenRect];
    NSPoint pt = [win.contentView convertPoint:winRect.origin fromView:nil];
    CGFloat viewH = win.contentView.bounds.size.height;
    CGFloat scale = win.backingScaleFactor > 0 ? win.backingScaleFactor : 1.0;

    int kind = 1;
    int button = 0;
    switch (event.type) {
        case NSEventTypeLeftMouseUp:
            kind = 3; button = 0; break;
        case NSEventTypeRightMouseUp:
            kind = 3; button = 1; break;
        case NSEventTypeOtherMouseUp:
            if (event.buttonNumber != 2) return;
            kind = 3; button = 2; break;
        case NSEventTypeLeftMouseDragged:
            kind = 1; button = 0; break;
        case NSEventTypeRightMouseDragged:
            kind = 1; button = 1; break;
        case NSEventTypeOtherMouseDragged:
            kind = 1; button = 2; break;
        case NSEventTypeMouseMoved:
            kind = 1; button = 0; break;
        default:
            return;
    }

    goDarwinOnMouse(del.winID, kind, pt.x, pt.y, viewH, scale, button, (uint64_t)event.modifierFlags);
    if (kind == 3) {
        [del releaseCaptureMonitors];
    }
}

void anyui_darwin_set_capture(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            AnyUIWindowDelegate *del = getDelegateForWindow(win);
            if (!del || del.localMonitor || del.globalMonitor) {
                return;
            }
            NSEventMask mask = NSEventMaskLeftMouseDragged |
                               NSEventMaskRightMouseDragged |
                               NSEventMaskOtherMouseDragged |
                               NSEventMaskLeftMouseUp |
                               NSEventMaskRightMouseUp |
                               NSEventMaskOtherMouseUp |
                               NSEventMaskMouseMoved;

            __weak NSWindow *weakWin = win;
            __weak AnyUIWindowDelegate *weakDel = del;

            del.localMonitor = [NSEvent addLocalMonitorForEventsMatchingMask:mask handler:^NSEvent *(NSEvent *event) {
                NSWindow *strongWin = weakWin;
                AnyUIWindowDelegate *strongDel = weakDel;
                if (!strongWin || !strongDel) return event;
                // Если событие уже адресовано нашему окну, оно обработается в AnyUIView без дублирования.
                if (event.window == strongWin) {
                    return event;
                }
                dispatchCapturedMouseEvent(strongWin, strongDel, event);
                return event;
            }];

            del.globalMonitor = [NSEvent addGlobalMonitorForEventsMatchingMask:mask handler:^(NSEvent *event) {
                NSWindow *strongWin = weakWin;
                AnyUIWindowDelegate *strongDel = weakDel;
                if (!strongWin || !strongDel) return;
                dispatchCapturedMouseEvent(strongWin, strongDel, event);
            }];
        }
    });
}

void anyui_darwin_release_capture(uintptr_t handle) {
    if (!handle) return;
    NSWindow *win = (__bridge NSWindow *)(void *)handle;
    runOnMainAsync(^{
        @autoreleasepool {
            AnyUIWindowDelegate *del = getDelegateForWindow(win);
            if (del) {
                [del releaseCaptureMonitors];
            }
        }
    });
}

#endif // __APPLE__
