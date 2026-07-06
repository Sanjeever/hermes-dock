#import <Cocoa/Cocoa.h>

extern void hermesDockTrayOpenApp(void);
extern void hermesDockTrayOpenWeb(void);
extern void hermesDockTrayCopyURL(void);
extern void hermesDockTrayQuitApp(void);

@interface HermesDockTrayTarget : NSObject
- (void)openApp:(id)sender;
- (void)openWeb:(id)sender;
- (void)copyURL:(id)sender;
- (void)quitApp:(id)sender;
@end

@implementation HermesDockTrayTarget
- (void)openApp:(id)sender { hermesDockTrayOpenApp(); }
- (void)openWeb:(id)sender { hermesDockTrayOpenWeb(); }
- (void)copyURL:(id)sender { hermesDockTrayCopyURL(); }
- (void)quitApp:(id)sender { hermesDockTrayQuitApp(); }
@end

static NSStatusItem *hermesDockStatusItem = nil;
static HermesDockTrayTarget *hermesDockTrayTarget = nil;

static NSMenuItem *hermesDockTrayItem(NSString *title, SEL action) {
    NSMenuItem *item = [[NSMenuItem alloc] initWithTitle:title action:action keyEquivalent:@""];
    [item setTarget:hermesDockTrayTarget];
    return item;
}

static NSImage *hermesDockTrayIcon(void) {
    NSImage *image = [[NSImage alloc] initWithSize:NSMakeSize(18, 18)];
    [image lockFocus];
    [[NSColor blackColor] setFill];

    NSBezierPath *left = [NSBezierPath bezierPathWithRoundedRect:NSMakeRect(4, 3, 3, 12) xRadius:1.5 yRadius:1.5];
    NSBezierPath *right = [NSBezierPath bezierPathWithRoundedRect:NSMakeRect(11, 3, 3, 12) xRadius:1.5 yRadius:1.5];
    NSBezierPath *middle = [NSBezierPath bezierPathWithRoundedRect:NSMakeRect(6, 8, 6, 2.5) xRadius:1.2 yRadius:1.2];
    [left fill];
    [right fill];
    [middle fill];

    [image unlockFocus];
    [image setTemplate:YES];
    return image;
}

void hermesDockSetupTray(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (hermesDockStatusItem != nil) {
            return;
        }
        hermesDockTrayTarget = [HermesDockTrayTarget new];
        hermesDockStatusItem = [[[NSStatusBar systemStatusBar] statusItemWithLength:NSSquareStatusItemLength] retain];
        hermesDockStatusItem.button.image = hermesDockTrayIcon();
        hermesDockStatusItem.button.imagePosition = NSImageOnly;
        hermesDockStatusItem.button.toolTip = @"企智盒";

        NSMenu *menu = [[NSMenu alloc] initWithTitle:@"企智盒"];
        [menu addItem:hermesDockTrayItem(@"打开企智盒", @selector(openApp:))];
        [menu addItem:hermesDockTrayItem(@"打开 Web 管理", @selector(openWeb:))];
        [menu addItem:hermesDockTrayItem(@"复制局域网地址", @selector(copyURL:))];
        [menu addItem:[NSMenuItem separatorItem]];
        [menu addItem:hermesDockTrayItem(@"退出", @selector(quitApp:))];
        hermesDockStatusItem.menu = menu;
    });
}

void hermesDockRemoveTray(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        if (hermesDockStatusItem != nil) {
            [[NSStatusBar systemStatusBar] removeStatusItem:hermesDockStatusItem];
            [hermesDockStatusItem release];
            hermesDockStatusItem = nil;
            [hermesDockTrayTarget release];
            hermesDockTrayTarget = nil;
        }
    });
}
