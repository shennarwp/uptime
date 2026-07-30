ALTER TABLE targets DROP COLUMN interval_s;
ALTER TABLE targets ADD COLUMN schedule TEXT NOT NULL DEFAULT '0 * * * * *';

DELETE FROM checks;
DELETE FROM targets;

INSERT INTO targets (id, name, url, schedule) VALUES
    (1, 'CV Staging', 'https://shennarwp.com/cv-staging/', '0 * * * * *'),
    (2, 'Azure Resume', 'https://resume.graysand-8af4e4eb.germanywestcentral.azurecontainerapps.io/', '0 * * * * *'),
    (3, 'Azure Static App', 'https://victorious-moss-02a509500.7.azurestaticapps.net/', '0 * * * * *'),
    (4, 'Shenna Piri', 'https://shenna.rwpiri.com', '0 * * * * *');
