/**
 * Browser Notification Service
 * 
 * Handles browser notification permissions and displays notifications
 * when new requests arrive via WebSocket.
 */

export type NotificationPermission = 'default' | 'granted' | 'denied';

export interface NotificationOptions {
  title: string;
  body?: string;
  icon?: string;
  badge?: string;
  tag?: string;
  requireInteraction?: boolean;
}

class BrowserNotificationService {
  private permission: NotificationPermission = 'default';
  private notificationIcon: string = '/images/logo-placeholder.png';

  /**
   * Check if browser notifications are supported
   */
  isSupported(): boolean {
    return 'Notification' in window;
  }

  /**
   * Get current notification permission status
   */
  getPermission(): NotificationPermission {
    if (!this.isSupported()) {
      return 'denied';
    }
    return Notification.permission as NotificationPermission;
  }

  /**
   * Request notification permission from the user
   */
  async requestPermission(): Promise<NotificationPermission> {
    if (!this.isSupported()) {
      console.warn('Browser notifications are not supported');
      return 'denied';
    }

    const currentPermission = Notification.permission;
    console.log(`[Notifications] Current permission status: ${currentPermission}`);

    if (currentPermission === 'granted') {
      this.permission = 'granted';
      console.log('[Notifications] Permission already granted');
      return 'granted';
    }

    if (currentPermission === 'denied') {
      this.permission = 'denied';
      console.warn('[Notifications] Permission was previously denied. To reset:');
      console.warn('  Firefox: about:preferences#privacy > Notifications > Settings > Remove localhost');
      console.warn('  Chrome: Settings > Privacy > Site Settings > Notifications > Remove localhost');
      return 'denied';
    }

    try {
      console.log('[Notifications] Requesting permission...');
      const permission = await Notification.requestPermission();
      this.permission = permission as NotificationPermission;
      console.log(`[Notifications] Permission result: ${permission}`);
      return this.permission;
    } catch (error) {
      console.error('Error requesting notification permission:', error);
      this.permission = 'denied';
      return 'denied';
    }
  }

  /**
   * Show a browser notification
   */
  showNotification(options: NotificationOptions): Notification | null {
    if (!this.isSupported()) {
      console.warn('[Notifications] Browser notifications are not supported');
      return null;
    }

    // Check permission again in case it changed
    const currentPermission = Notification.permission;
    if (currentPermission !== 'granted') {
      console.warn(`[Notifications] Permission not granted (current: ${currentPermission})`);
      if (currentPermission === 'denied') {
        console.warn('[Notifications] To enable: Reset notification permission for this site in browser settings');
      }
      return null;
    }

    try {
      const notification = new Notification(options.title, {
        body: options.body,
        icon: options.icon || this.notificationIcon,
        badge: options.badge || this.notificationIcon,
        tag: options.tag,
        requireInteraction: options.requireInteraction || false,
      });

      // Focus window when notification is clicked
      notification.onclick = () => {
        window.focus();
        notification.close();
      };

      // Auto-close after 5 seconds if not requiring interaction
      if (!options.requireInteraction) {
        setTimeout(() => {
          notification.close();
        }, 5000);
      }

      return notification;
    } catch (error) {
      console.error('Error showing notification:', error);
      return null;
    }
  }

  /**
   * Show a notification for a new request
   */
  showRequestNotification(requestTitle: string, requestType: string): Notification | null {
    const typeLabels: Record<string, string> = {
      confirm: 'Confirmation Request',
      select: 'Selection Request',
      form: 'Form Request',
      upload: 'Upload Request',
      table: 'Table Request',
    };

    const typeLabel = typeLabels[requestType] || 'New Request';

    return this.showNotification({
      title: typeLabel,
      body: requestTitle,
      tag: 'plz-confirm-request',
      requireInteraction: true, // Keep notification until user interacts
    });
  }
}

// Export singleton instance
export const browserNotificationService = new BrowserNotificationService();

