//go:build darwin && cgo

#if defined(__APPLE__) && defined(__MACH__)

#import <Cocoa/Cocoa.h>
#import <UserNotifications/UserNotifications.h>
#include <stdlib.h>
#include <string.h>
#include "darwin_bridge.h"

// Обратные вызовы в Go (экспортируются в darwin_tray.go).
extern void goDarwinOnTrayClick(uint64_t trayID);
extern void goDarwinOnTrayDoubleClick(uint64_t trayID);
extern void goDarwinOnTrayRightClick(uint64_t trayID);
extern void goDarwinOnTrayMenuItem(uint64_t trayID, int itemID);
extern void goDarwinOnNotificationClick(void);

static void sysRunOnMainSync(dispatch_block_t block) {
    if ([NSThread isMainThread]) {
        block();
    } else {
        dispatch_sync(dispatch_get_main_queue(), block);
    }
}

static void sysRunOnMainAsync(dispatch_block_t block) {
    if ([NSThread isMainThread]) {
        block();
    } else {
        dispatch_async(dispatch_get_main_queue(), block);
    }
}

static char* sysStrdupNSString(NSString *str) {
    if (!str) return NULL;
    const char *utf8 = [str UTF8String];
    if (!utf8) return NULL;
    return strdup(utf8);
}

static NSArray<NSString *>* sysParseExtensionsCSV(const char* csv) {
    if (!csv || csv[0] == '\0') return nil;
    NSString *raw = [NSString stringWithUTF8String:csv];
    if (!raw || raw.length == 0) return nil;
    NSArray<NSString *> *parts = [raw componentsSeparatedByString:@","];
    NSMutableArray<NSString *> *res = [NSMutableArray arrayWithCapacity:parts.count];
    for (NSString *p in parts) {
        NSString *trimmed = [p stringByTrimmingCharactersInSet:[NSCharacterSet whitespaceAndNewlineCharacterSet]];
        if (trimmed.length > 0) {
            [res addObject:trimmed];
        }
    }
    return res.count > 0 ? res : nil;
}

// ============================================================================
// Системный трей (NSStatusItem)
// ============================================================================

@interface AnyUITrayTarget : NSObject
@property (nonatomic, assign) uint64_t trayID;
@property (nonatomic, strong) NSStatusItem *statusItem;
@property (nonatomic, strong) NSMenu *menu;
@end

@implementation AnyUITrayTarget

- (void)onStatusButtonClick:(id)sender {
    (void)sender;
    NSEvent *ev = [NSApp currentEvent];
    if (ev && (ev.type == NSEventTypeRightMouseUp ||
               (ev.type == NSEventTypeLeftMouseUp && (ev.modifierFlags & NSEventModifierFlagControl)))) {
        goDarwinOnTrayRightClick(self.trayID);
        return;
    }
    if (ev && ev.clickCount >= 2) {
        goDarwinOnTrayDoubleClick(self.trayID);
        return;
    }
    goDarwinOnTrayClick(self.trayID);
}

- (void)onMenuItemAction:(NSMenuItem *)sender {
    goDarwinOnTrayMenuItem(self.trayID, (int)sender.tag);
}

@end

uintptr_t anyui_darwin_tray_create(uint64_t trayID, const char* tooltip) {
    __block uintptr_t result = 0;
    NSString *tip = tooltip ? [NSString stringWithUTF8String:tooltip] : nil;

    sysRunOnMainSync(^{
        @autoreleasepool {
            [NSApplication sharedApplication];
            NSStatusItem *item = [[NSStatusBar systemStatusBar] statusItemWithLength:NSVariableStatusItemLength];
            if (!item) {
                return;
            }
            AnyUITrayTarget *target = [[AnyUITrayTarget alloc] init];
            target.trayID = trayID;
            target.statusItem = item;
            target.menu = [[NSMenu alloc] init];
            target.menu.autoenablesItems = NO;

            NSStatusBarButton *btn = item.button;
            if (btn) {
                btn.target = target;
                btn.action = @selector(onStatusButtonClick:);
                [btn sendActionOn:(NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp)];
                btn.title = @"●";
                if (tip && tip.length > 0) {
                    btn.toolTip = tip;
                }
            }
            result = (uintptr_t)CFBridgingRetain(target);
        }
    });

    return result;
}

void anyui_darwin_tray_destroy(uintptr_t handle) {
    if (!handle) return;
    sysRunOnMainAsync(^{
        @autoreleasepool {
            AnyUITrayTarget *target = (AnyUITrayTarget *)CFBridgingRelease((CFTypeRef)handle);
            if (target.statusItem) {
                [[NSStatusBar systemStatusBar] removeStatusItem:target.statusItem];
                target.statusItem = nil;
            }
            target.menu = nil;
        }
    });
}

void anyui_darwin_tray_set_tooltip(uintptr_t handle, const char* tooltip) {
    if (!handle) return;
    AnyUITrayTarget *target = (__bridge AnyUITrayTarget *)(void *)handle;
    NSString *tip = tooltip ? [NSString stringWithUTF8String:tooltip] : @"";
    sysRunOnMainAsync(^{
        @autoreleasepool {
            if (target.statusItem && target.statusItem.button) {
                target.statusItem.button.toolTip = tip;
            }
        }
    });
}

void anyui_darwin_tray_set_icon(uintptr_t handle, const void* rgbaPixels, int width, int height) {
    if (!handle || !rgbaPixels || width <= 0 || height <= 0) return;
    AnyUITrayTarget *target = (__bridge AnyUITrayTarget *)(void *)handle;
    size_t byteCount = (size_t)width * (size_t)height * 4;
    NSData *pixelCopy = [NSData dataWithBytes:rgbaPixels length:byteCount];

    sysRunOnMainAsync(^{
        @autoreleasepool {
            if (!target.statusItem || !target.statusItem.button) return;
            NSBitmapImageRep *rep = [[NSBitmapImageRep alloc]
                initWithBitmapDataPlanes:NULL
                              pixelsWide:width
                              pixelsHigh:height
                           bitsPerSample:8
                         samplesPerPixel:4
                                hasAlpha:YES
                                isPlanar:NO
                          colorSpaceName:NSDeviceRGBColorSpace
                            bitmapFormat:NSBitmapFormatAlphaNonpremultiplied
                             bytesPerRow:width * 4
                            bitsPerPixel:32];
            if (!rep || !rep.bitmapData) return;
            memcpy(rep.bitmapData, pixelCopy.bytes, byteCount);

            NSImage *img = [[NSImage alloc] initWithSize:NSMakeSize(18, 18)];
            [img addRepresentation:rep];
            // Шаблонный режим позволяет macOS автоматически адаптировать монохромную иконку под тему панели меню.
            [img setTemplate:YES];

            target.statusItem.button.title = @"";
            target.statusItem.button.image = img;
            target.statusItem.button.imageScaling = NSImageScaleProportionallyDown;
        }
    });
}

void anyui_darwin_tray_clear_menu(uintptr_t handle) {
    if (!handle) return;
    AnyUITrayTarget *target = (__bridge AnyUITrayTarget *)(void *)handle;
    sysRunOnMainAsync(^{
        @autoreleasepool {
            [target.menu removeAllItems];
        }
    });
}

void anyui_darwin_tray_add_menu_item(uintptr_t handle, int itemID, const char* title, int separator, int checked, int disabled) {
    if (!handle) return;
    AnyUITrayTarget *target = (__bridge AnyUITrayTarget *)(void *)handle;
    NSString *titleStr = title ? [NSString stringWithUTF8String:title] : @"";

    sysRunOnMainAsync(^{
        @autoreleasepool {
            if (separator) {
                [target.menu addItem:[NSMenuItem separatorItem]];
                return;
            }
            NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:titleStr
                                                          action:@selector(onMenuItemAction:)
                                                   keyEquivalent:@""];
            item.target = target;
            item.tag = itemID;
            item.state = checked ? NSControlStateValueOn : NSControlStateValueOff;
            item.enabled = (disabled == 0);
            [target.menu addItem:item];
        }
    });
}

void anyui_darwin_tray_popup_menu(uintptr_t handle) {
    if (!handle) return;
    AnyUITrayTarget *target = (__bridge AnyUITrayTarget *)(void *)handle;
    sysRunOnMainAsync(^{
        @autoreleasepool {
            if (!target.statusItem || !target.menu || target.menu.numberOfItems == 0) {
                return;
            }
            target.statusItem.menu = target.menu;
            [target.statusItem.button performClick:nil];
            target.statusItem.menu = nil;
        }
    });
}

// ============================================================================
// Нативные уведомления (UNUserNotificationCenter)
// ============================================================================

@interface AnyUINotificationDelegate : NSObject <UNUserNotificationCenterDelegate>
@end

@implementation AnyUINotificationDelegate

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
       willPresentNotification:(UNNotification *)notification
         withCompletionHandler:(void (^)(UNNotificationPresentationOptions options))completionHandler {
    (void)center;
    (void)notification;
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
    completionHandler(UNNotificationPresentationOptionAlert | UNNotificationPresentationOptionSound);
#pragma clang diagnostic pop
}

- (void)userNotificationCenter:(UNUserNotificationCenter *)center
didReceiveNotificationResponse:(UNNotificationResponse *)response
         withCompletionHandler:(void (^)(void))completionHandler {
    (void)center;
    (void)response;
    goDarwinOnNotificationClick();
    completionHandler();
}

@end

static AnyUINotificationDelegate *gNotificationDelegate = nil;

int anyui_darwin_notify_un(const char* title, const char* message, int silent) {
    @autoreleasepool {
        // UNUserNotificationCenter требует валидный bundleIdentifier (.app bundle).
        NSString *bundleID = [[NSBundle mainBundle] bundleIdentifier];
        if (!bundleID || bundleID.length == 0) {
            return 0;
        }
        if (!NSClassFromString(@"UNUserNotificationCenter")) {
            return 0;
        }

        UNUserNotificationCenter *center = [UNUserNotificationCenter currentNotificationCenter];
        if (!center) {
            return 0;
        }

        if (!gNotificationDelegate) {
            gNotificationDelegate = [[AnyUINotificationDelegate alloc] init];
            center.delegate = gNotificationDelegate;
        }

        NSString *titleStr = title ? [NSString stringWithUTF8String:title] : @"";
        NSString *msgStr = message ? [NSString stringWithUTF8String:message] : @"";

        [center requestAuthorizationWithOptions:(UNAuthorizationOptionAlert | UNAuthorizationOptionSound)
                              completionHandler:^(BOOL granted, NSError * _Nullable error) {
            (void)error;
            if (!granted) {
                return;
            }
            UNMutableNotificationContent *content = [[UNMutableNotificationContent alloc] init];
            content.title = titleStr ? titleStr : @"";
            content.body = msgStr ? msgStr : @"";
            if (!silent) {
                content.sound = [UNNotificationSound defaultSound];
            }
            NSString *reqID = [[NSUUID UUID] UUIDString];
            UNNotificationRequest *req = [UNNotificationRequest requestWithIdentifier:reqID
                                                                              content:content
                                                                              trigger:nil];
            [center addNotificationRequest:req withCompletionHandler:nil];
        }];
        return 1;
    }
}

// ============================================================================
// Нативные диалоги (NSOpenPanel, NSSavePanel, NSAlert)
// ============================================================================

char* anyui_darwin_open_file_dialog(
    const char* title,
    const char* initialDir,
    const char* extensionsCSV,
    int allowMultiple,
    int chooseDirectories
) {
    __block char *result = NULL;
    NSString *titleStr = title ? [NSString stringWithUTF8String:title] : nil;
    NSString *dirStr = initialDir ? [NSString stringWithUTF8String:initialDir] : nil;
    NSArray<NSString *> *exts = sysParseExtensionsCSV(extensionsCSV);

    sysRunOnMainSync(^{
        @autoreleasepool {
            [NSApplication sharedApplication];
            [NSApp activateIgnoringOtherApps:YES];

            NSOpenPanel *panel = [NSOpenPanel openPanel];
            panel.canChooseFiles = (chooseDirectories == 0);
            panel.canChooseDirectories = (chooseDirectories != 0);
            panel.allowsMultipleSelection = (allowMultiple != 0);
            panel.resolvesAliases = YES;

            if (titleStr && titleStr.length > 0) {
                panel.title = titleStr;
                panel.message = titleStr;
            }
            if (dirStr && dirStr.length > 0) {
                panel.directoryURL = [NSURL fileURLWithPath:dirStr];
            }
            if (!chooseDirectories && exts && exts.count > 0) {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
                panel.allowedFileTypes = exts;
#pragma clang diagnostic pop
            }

            if ([panel runModal] == NSModalResponseOK) {
                NSMutableArray<NSString *> *paths = [NSMutableArray arrayWithCapacity:panel.URLs.count];
                for (NSURL *u in panel.URLs) {
                    if (u.path && u.path.length > 0) {
                        [paths addObject:u.path];
                    }
                }
                if (paths.count > 0) {
                    result = sysStrdupNSString([paths componentsJoinedByString:@"\n"]);
                }
            }
        }
    });

    return result;
}

char* anyui_darwin_save_file_dialog(
    const char* title,
    const char* initialDir,
    const char* defaultName,
    const char* extensionsCSV
) {
    __block char *result = NULL;
    NSString *titleStr = title ? [NSString stringWithUTF8String:title] : nil;
    NSString *dirStr = initialDir ? [NSString stringWithUTF8String:initialDir] : nil;
    NSString *nameStr = defaultName ? [NSString stringWithUTF8String:defaultName] : nil;
    NSArray<NSString *> *exts = sysParseExtensionsCSV(extensionsCSV);

    sysRunOnMainSync(^{
        @autoreleasepool {
            [NSApplication sharedApplication];
            [NSApp activateIgnoringOtherApps:YES];

            NSSavePanel *panel = [NSSavePanel savePanel];
            panel.canCreateDirectories = YES;

            if (titleStr && titleStr.length > 0) {
                panel.title = titleStr;
                panel.message = titleStr;
            }
            if (dirStr && dirStr.length > 0) {
                panel.directoryURL = [NSURL fileURLWithPath:dirStr];
            }
            if (nameStr && nameStr.length > 0) {
                panel.nameFieldStringValue = nameStr;
            }
            if (exts && exts.count > 0) {
#pragma clang diagnostic push
#pragma clang diagnostic ignored "-Wdeprecated-declarations"
                panel.allowedFileTypes = exts;
#pragma clang diagnostic pop
            }

            if ([panel runModal] == NSModalResponseOK && panel.URL && panel.URL.path) {
                result = sysStrdupNSString(panel.URL.path);
            }
        }
    });

    return result;
}

int anyui_darwin_message_box(const char* title, const char* text, int style, int buttons) {
    __block int result = 0; // ResNone
    NSString *titleStr = title ? [NSString stringWithUTF8String:title] : @"";
    NSString *textStr = text ? [NSString stringWithUTF8String:text] : @"";

    sysRunOnMainSync(^{
        @autoreleasepool {
            [NSApplication sharedApplication];
            [NSApp activateIgnoringOtherApps:YES];

            NSAlert *alert = [[NSAlert alloc] init];
            alert.messageText = titleStr ? titleStr : @"";
            alert.informativeText = textStr ? textStr : @"";

            switch (style) {
                case 1: // MsgWarning
                    alert.alertStyle = NSAlertStyleWarning;
                    break;
                case 2: // MsgError
                    alert.alertStyle = NSAlertStyleCritical;
                    break;
                default: // MsgInfo, MsgQuestion
                    alert.alertStyle = NSAlertStyleInformational;
                    break;
            }

            // Порядок кнопок в NSAlert: первая добавленная кнопка — правая (по умолчанию).
            switch (buttons) {
                case 1: // MsgOKCancel
                    [alert addButtonWithTitle:@"OK"];
                    [alert addButtonWithTitle:@"Cancel"];
                    break;
                case 2: // MsgYesNo
                    [alert addButtonWithTitle:@"Yes"];
                    [alert addButtonWithTitle:@"No"];
                    break;
                case 3: // MsgYesNoCancel
                    [alert addButtonWithTitle:@"Yes"];
                    [alert addButtonWithTitle:@"No"];
                    [alert addButtonWithTitle:@"Cancel"];
                    break;
                default: // MsgOK
                    [alert addButtonWithTitle:@"OK"];
                    break;
            }

            NSModalResponse resp = [alert runModal];
            if (buttons == 1) { // MsgOKCancel
                result = (resp == NSAlertFirstButtonReturn) ? 1 : 2; // ResOK : ResCancel
            } else if (buttons == 2) { // MsgYesNo
                result = (resp == NSAlertFirstButtonReturn) ? 3 : 4; // ResYes : ResNo
            } else if (buttons == 3) { // MsgYesNoCancel
                if (resp == NSAlertFirstButtonReturn) result = 3;      // ResYes
                else if (resp == NSAlertSecondButtonReturn) result = 4; // ResNo
                else result = 2;                                        // ResCancel
            } else { // MsgOK
                result = 1; // ResOK
            }
        }
    });

    return result;
}

// ============================================================================
// Буфер обмена (NSPasteboard)
// ============================================================================

char* anyui_darwin_clipboard_get_text(void) {
    __block char *result = NULL;
    sysRunOnMainSync(^{
        @autoreleasepool {
            NSPasteboard *pb = [NSPasteboard generalPasteboard];
            NSString *str = [pb stringForType:NSPasteboardTypeString];
            if (str) {
                result = sysStrdupNSString(str);
            }
        }
    });
    return result;
}

int anyui_darwin_clipboard_set_text(const char* text) {
    __block int ok = 0;
    NSString *str = text ? [NSString stringWithUTF8String:text] : @"";
    sysRunOnMainSync(^{
        @autoreleasepool {
            NSPasteboard *pb = [NSPasteboard generalPasteboard];
            [pb clearContents];
            ok = [pb setString:(str ? str : @"") forType:NSPasteboardTypeString] ? 1 : 0;
        }
    });
    return ok;
}

void* anyui_darwin_clipboard_get_image(int* outLen) {
    if (outLen) *outLen = 0;
    __block void *result = NULL;
    __block int len = 0;

    sysRunOnMainSync(^{
        @autoreleasepool {
            NSPasteboard *pb = [NSPasteboard generalPasteboard];
            NSData *pngData = [pb dataForType:NSPasteboardTypePNG];
            if (!pngData || pngData.length == 0) {
                NSData *tiffData = [pb dataForType:NSPasteboardTypeTIFF];
                if (tiffData && tiffData.length > 0) {
                    NSBitmapImageRep *rep = [NSBitmapImageRep imageRepWithData:tiffData];
                    if (rep) {
                        pngData = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
                    }
                }
            }
            if (pngData && pngData.length > 0) {
                len = (int)pngData.length;
                result = malloc((size_t)len);
                if (result) {
                    memcpy(result, pngData.bytes, (size_t)len);
                } else {
                    len = 0;
                }
            }
        }
    });

    if (outLen) *outLen = len;
    return result;
}

int anyui_darwin_clipboard_set_image(const void* pngData, int len) {
    if (!pngData || len <= 0) return 0;
    __block int ok = 0;
    NSData *data = [NSData dataWithBytes:pngData length:(size_t)len];

    sysRunOnMainSync(^{
        @autoreleasepool {
            NSPasteboard *pb = [NSPasteboard generalPasteboard];
            [pb clearContents];
            ok = [pb setData:data forType:NSPasteboardTypePNG] ? 1 : 0;
        }
    });
    return ok;
}

#endif // __APPLE__ && __MACH__

