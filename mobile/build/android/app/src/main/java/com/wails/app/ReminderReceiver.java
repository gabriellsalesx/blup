package com.wails.app;

import android.app.AlarmManager;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.media.AudioAttributes;
import android.media.RingtoneManager;
import android.net.Uri;
import android.os.Build;

import androidx.core.app.NotificationCompat;

/**
 * Shows a reminder or rings an alarm, and puts them back after the phone
 * restarts. An alarm rings, with the alarm volume, until it is stopped,
 * snoozed, opened or swiped away.
 */
public class ReminderReceiver extends BroadcastReceiver {
    private static final String CHANNEL = "blup_reminders";
    private static final String ALARMS = "blup_alarms";
    static final String RING = "io.getblup.notes.RING";
    private static final String STOP = "io.getblup.notes.STOP";
    private static final String SNOOZE = "io.getblup.notes.SNOOZE";
    private static final long SNOOZE_MS = 10 * 60 * 1000L;

    @Override
    public void onReceive(Context c, Intent intent) {
        String action = intent.getAction();
        if (Intent.ACTION_BOOT_COMPLETED.equals(action) || Intent.ACTION_MY_PACKAGE_REPLACED.equals(action)) {
            ReminderScheduler.schedule(c);
            return;
        }
        NotificationManager nm = (NotificationManager) c.getSystemService(Context.NOTIFICATION_SERVICE);
        int id = intent.getIntExtra("id", 1);
        if (STOP.equals(action)) {
            nm.cancel(id);
            return;
        }
        if (SNOOZE.equals(action)) {
            nm.cancel(id);
            snooze(c, intent);
            return;
        }
        Intent open = new Intent(c, MainActivity.class).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK | Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent tap = PendingIntent.getActivity(c, 0, open, PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
        boolean alarm = intent.getBooleanExtra("alarm", false);
        if (alarm) alarmChannel(nm, label(intent, "alarms", "Alarmes"));
        else nm.createNotificationChannel(new NotificationChannel(CHANNEL, label(intent, "channel", "Lembretes"), NotificationManager.IMPORTANCE_HIGH));
        NotificationCompat.Builder b = new NotificationCompat.Builder(c, alarm ? ALARMS : CHANNEL)
                .setSmallIcon(R.drawable.ic_notify)
                .setContentTitle(intent.getStringExtra("title"))
                .setContentText(intent.getStringExtra("body"))
                .setContentIntent(tap)
                .setAutoCancel(true)
                .setPriority(NotificationCompat.PRIORITY_HIGH);
        if (alarm) {
            b.setCategory(NotificationCompat.CATEGORY_ALARM)
                    .addAction(0, label(intent, "stop", "Parar"), action(c, STOP, intent))
                    .addAction(0, label(intent, "snooze", "Adiar 10 minutos"), action(c, SNOOZE, intent));
        }
        Notification n = b.build();
        if (alarm) n.flags |= Notification.FLAG_INSISTENT; // the sound repeats
        nm.notify(id, n);
    }

    /** The alarms' channel: the alarm sound, at the alarm volume, with vibration. */
    private static void alarmChannel(NotificationManager nm, String name) {
        NotificationChannel ch = new NotificationChannel(ALARMS, name, NotificationManager.IMPORTANCE_HIGH);
        Uri sound = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_ALARM);
        if (sound == null) sound = RingtoneManager.getDefaultUri(RingtoneManager.TYPE_NOTIFICATION);
        ch.setSound(sound, new AudioAttributes.Builder()
                .setUsage(AudioAttributes.USAGE_ALARM)
                .setContentType(AudioAttributes.CONTENT_TYPE_SONIFICATION)
                .build());
        ch.enableVibration(true);
        ch.setVibrationPattern(new long[]{0, 800, 600, 800, 600});
        ch.setLockscreenVisibility(Notification.VISIBILITY_PUBLIC);
        nm.createNotificationChannel(ch);
    }

    /** A text in the app's language, sent with the reminder; Portuguese for an older reminder. */
    private static String label(Intent i, String key, String fallback) {
        String s = i.getStringExtra(key);
        return s == null || s.isEmpty() ? fallback : s;
    }

    private static PendingIntent action(Context c, String what, Intent from) {
        Intent i = new Intent(c, ReminderReceiver.class).setAction(what).putExtras(from);
        return PendingIntent.getBroadcast(c, from.getIntExtra("id", 1), i,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
    }

    /** Rings the same alarm again in 10 minutes. */
    private static void snooze(Context c, Intent from) {
        Intent i = new Intent(c, ReminderReceiver.class).setAction(RING).putExtras(from);
        PendingIntent pi = PendingIntent.getBroadcast(c, from.getIntExtra("id", 1), i,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
        AlarmManager am = (AlarmManager) c.getSystemService(Context.ALARM_SERVICE);
        long at = System.currentTimeMillis() + SNOOZE_MS;
        if (Build.VERSION.SDK_INT >= 31 && !am.canScheduleExactAlarms()) {
            am.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pi);
        } else {
            am.setExactAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pi);
        }
    }
}
