DELETE FROM checks;
DELETE FROM targets;

INSERT INTO targets (id, name, url, interval_s) VALUES
    (1, 'CV Staging', 'https://shennarwp.com/cv-staging/', 60),
    (2, 'Azure Resume', 'https://resume.graysand-8af4e4eb.germanywestcentral.azurecontainerapps.io/', 60),
    (3, 'Azure Static App', 'https://victorious-moss-02a509500.7.azurestaticapps.net/', 60),
    (4, 'Shenna Piri', 'https://shenna.rwpiri.com', 60);
