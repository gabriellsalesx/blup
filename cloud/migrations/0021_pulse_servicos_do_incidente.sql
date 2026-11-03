CREATE TABLE `pulse_incident_component` (
	`incident_id` text NOT NULL,
	`monitor_id` text NOT NULL,
	`status` text NOT NULL,
	PRIMARY KEY(`incident_id`, `monitor_id`)
);
