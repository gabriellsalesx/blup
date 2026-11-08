package com.wails.app;

import android.app.AlarmManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.os.Build;
import android.util.Log;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.File;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.util.HashSet;
import java.util.Set;

/**
 * Hands the coming reminders (files/reminders.json, written by the Go side)
 * to the system, which fires them with the app minimized or closed.
 */
final class ReminderScheduler {
    private static final String TAG = "BlupReminders";

    private ReminderScheduler() {}

    /** The same alarm as intent() made, to cancel it: the system matches by id, not by extras. */
    private static PendingIntent cancelling(Context c, int id) {
        return PendingIntent.getBroadcast(c, id, new Intent(c, ReminderReceiver.class),
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
    }

    private static PendingIntent intent(Context c, int id, JSONObject r) throws org.json.JSONException {
        // The texts in the app's language come with each reminder (Go writes them).
        Intent i = new Intent(c, ReminderReceiver.class)
                .putExtra("id", id).putExtra("title", r.getString("title")).putExtra("body", r.getString("body"))
                .putExtra("alarm", r.optBoolean("alarm", false))
                .putExtra("stop", r.optString("stop")).putExtra("snooze", r.optString("snooze"))
                .putExtra("channel", r.optString("channel")).putExtra("alarms", r.optString("alarms"));
        return PendingIntent.getBroadcast(c, id, i,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
    }

    static boolean hasReminders(Context c) {
        File f = new File(c.getFilesDir(), "reminders.json");
        return f.length() > 2;
    }

    static void schedule(Context c) {
        File f = new File(c.getFilesDir(), "reminders.json");
        if (!f.exists()) return;
        try {
            JSONArray list = new JSONArray(new String(Files.readAllBytes(f.toPath()), StandardCharsets.UTF_8));
            AlarmManager am = (AlarmManager) c.getSystemService(Context.ALARM_SERVICE);
            SharedPreferences prefs = c.getSharedPreferences("reminders", Context.MODE_PRIVATE);
            Set<String> old = new HashSet<>(prefs.getStringSet("ids", new HashSet<>()));
            Set<String> ids = new HashSet<>();
            long now = System.currentTimeMillis();
            for (int i = 0; i < list.length(); i++) {
                JSONObject r = list.getJSONObject(i);
                long at = r.getLong("at");
                if (at <= now) continue;
                int id = r.getInt("id");
                PendingIntent pi = intent(c, id, r);
                if (Build.VERSION.SDK_INT >= 31 && !am.canScheduleExactAlarms()) {
                    am.setAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pi);
                } else {
                    am.setExactAndAllowWhileIdle(AlarmManager.RTC_WAKEUP, at, pi);
                }
                ids.add(String.valueOf(id));
            }
            old.removeAll(ids);
            for (String id : old) { // deleted or moved events
                am.cancel(cancelling(c, Integer.parseInt(id)));
            }
            prefs.edit().putStringSet("ids", ids).apply();
        } catch (Exception e) {
            Log.e(TAG, "schedule failed", e);
        }
    }
}
