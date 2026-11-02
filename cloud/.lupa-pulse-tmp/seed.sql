INSERT INTO user(id,name,email,email_verified) VALUES('pulse-sdk-user','SDK test','sdk@audit.invalid',1);
INSERT INTO subscription(id,plan,reference_id,status) VALUES('pulse-sdk-sub','ultra_pulse','pulse-sdk-user','active');
INSERT INTO session(id,token,user_id,expires_at,created_at,updated_at) VALUES('pulse-sdk-session','pulse-sdk-local-session','pulse-sdk-user',4102444800000,1700000000000,1700000000000);
