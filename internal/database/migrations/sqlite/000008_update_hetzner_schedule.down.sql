UPDATE targets SET schedule = '0 * * * * *' WHERE name = 'Hetzner Staging';
UPDATE targets SET schedule = '0 0 */6 * * *' WHERE name = 'Azure Static Web';
UPDATE targets SET schedule = '0 0 */6 * * *' WHERE name = 'Oracle Prod';
