
-- Table Definition
CREATE TABLE logins (
    id uuid PRIMARY KEY NOT NULL,
    user_id uuid NOT NULL,
    created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE tokens (
    id	UUID PRIMARY KEY,
    user_id	UUID REFERENCES users(id),
    name	TEXT,
    value	TEXT,
    counter	int4,
    created_at	timestamptz NOT NULL DEFAULT NOW(),
    updated_at	timestamptz NOT NULL DEFAULT NOW()
);


CREATE TABLE users (
    id UUID PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT,
    password TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'UNCONFIRMED',
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    ssn VARCHAR(12),
    sms_opt_in BOOLEAN NOT NULL DEFAULT FALSE,
    active_role TEXT NOT NULL DEFAULT 'STUDENT',

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tutors (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id),

    image TEXT NOT NULL,
    location_id INT REFERENCES locations(id),
    online_lessons BOOLEAN NOT NULL DEFAULT FALSE,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);



CREATE TABLE tutor_subjects (
    tutor_id	UUID REFERENCES tutors(id),
    subject_id	int REFERENCES subjects(id),
    PRIMARY KEY (tutor_id, subject_id)
);

CREATE TABLE tutor_levels (
    tutor_id	UUID REFERENCES tutors(id),
    level_id	int REFERENCES levels(id),
    PRIMARY KEY (tutor_id, level_id)
);

CREATE TABLE tutor_locations (
    tutor_id	UUID REFERENCES tutors(id),
    location_id	int REFERENCES locations(id),
    PRIMARY KEY (tutor_id, location_id)
);

CREATE TABLE logins (
    id	UUID PRIMARY KEY,
    user_id	UUID REFERENCES users(id),
    created_at	timestamptz NOT NULL DEFAULT NOW(),
    updated_at	timestamptz NOT NULL DEFAULT NOW()
);

--- mails
CREATE TABLE mails (
    id SERIAL PRIMARY KEY,
    "to" TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    status TEXT
);


CREATE TABLE locations (
    id int SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE subjects (
    id int SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE levels (
    id int SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);


CREATE TABLE lessons (
    id UUID PRIMARY KEY,
    student_id UUID REFERENCES users(id),
    subject_id INT REFERENCES subjects(id),
    level_id INT REFERENCES levels(id),
    location_id INT REFERENCES locations(id),
    online_lessson BOOLEAN NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    start_at TIMESTAMPTZ NOT NULL,
    tutor_id UUID REFERENCES tutors(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE lesson_requests (
    id SERIAL PRIMARY KEY,
    lesson_id UUID REFERENCES lessons(id),
    tutor_id UUID REFERENCES tutors(id),
    status TEXT NOT NULL DEFAULT 'PENDING',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE ratings (
    id SERIAL PRIMARY KEY,
    lesson_id UUID REFERENCES lessons(id),
    tutor_id UUID REFERENCES tutors(id),
    grade INT NOT NULL DEFAULT 0,
    feedback TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO LOCATIONS (id,name) values (1,	'Stockholm');
INSERT INTO LOCATIONS (id,name) values (2,	'Göteborg');
INSERT INTO LOCATIONS (id,name) values (3,	'Malmö');
INSERT INTO LOCATIONS (id,name) values (4,	'Blekinge');
INSERT INTO LOCATIONS (id,name) values (5,	'Bohuslän');
INSERT INTO LOCATIONS (id,name) values (6,	'Dalarna');
INSERT INTO LOCATIONS (id,name) values (7,	'Dalsland');
INSERT INTO LOCATIONS (id,name) values (8,	'Gotland');
INSERT INTO LOCATIONS (id,name) values (9,	'Gästrikland');
INSERT INTO LOCATIONS (id,name) values (10,	'Halland');
INSERT INTO LOCATIONS (id,name) values (11,	'Hälsingland');
INSERT INTO LOCATIONS (id,name) values (12,	'Härjedalen');
INSERT INTO LOCATIONS (id,name) values (13,	'Jämtland');
INSERT INTO LOCATIONS (id,name) values (14,	'Lappland');
INSERT INTO LOCATIONS (id,name) values (15,	'Medelpad');
INSERT INTO LOCATIONS (id,name) values (16,	'Norrbotten');
INSERT INTO LOCATIONS (id,name) values (17,	'Närke');
INSERT INTO LOCATIONS (id,name) values (18,	'Skåne');
INSERT INTO LOCATIONS (id,name) values (19,	'Småland');
INSERT INTO LOCATIONS (id,name) values (20,	'Södermanland');
INSERT INTO LOCATIONS (id,name) values (21,	'Uppland');
INSERT INTO LOCATIONS (id,name) values (22,	'Värmland');
INSERT INTO LOCATIONS (id,name) values (23,	'Västerbotten');
INSERT INTO LOCATIONS (id,name) values (24,	'Västergötland');
INSERT INTO LOCATIONS (id,name) values (25,	'Västmanland');
INSERT INTO LOCATIONS (id,name) values (26,	'Ångermanland');
INSERT INTO LOCATIONS (id,name) values (27,	'Öland');
INSERT INTO LOCATIONS (id,name) values (28,	'Östergötland');
INSERT INTO LOCATIONS (id,name) values (29,	'Åland');


INSERT INTO subjects (id,name) values (1,'Biologi');
INSERT INTO subjects (id,name) values (2,'Ekonomi');
INSERT INTO subjects (id,name) values (3,'Engelska');
INSERT INTO subjects (id,name) values (4,'Franska');
INSERT INTO subjects (id,name) values (5,'Fysik');
INSERT INTO subjects (id,name) values (6,'Historia');
INSERT INTO subjects (id,name) values (7,'Italienska');
INSERT INTO subjects (id,name) values (8,'Juridik');
INSERT INTO subjects (id,name) values (9,'Kemi');
INSERT INTO subjects (id,name) values (10,'Matematik');
INSERT INTO subjects (id,name) values (11,'Nationella prov');
INSERT INTO subjects (id,name) values (12,'Programmering');
INSERT INTO subjects (id,name) values (13,'Psykologi');
INSERT INTO subjects (id,name) values (14,'Samhällskunskap');
INSERT INTO subjects (id,name) values (15,'Spanska');
INSERT INTO subjects (id,name) values (16,'Svenska');
INSERT INTO subjects (id,name) values (17,'Tyska');
INSERT INTO subjects (id,name) values (18,'Övriga språk');


INSERT INTO levels (id,name) values (1,'Lågstadiet');
INSERT INTO levels (id,name) values (2,'Mellanstadiet');
INSERT INTO levels (id,name) values (3,'Högstadiet');
INSERT INTO levels (id,name) values (4,'Gymnasiet');
INSERT INTO levels (id,name) values (5,'Universitet/Högskola');
INSERT INTO levels (id,name) values (6,'Vuxenutbildning');