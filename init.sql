CREATE TABLE sessions (
    id INT AUTO_INCREMENT PRIMARY KEY,
    host VARCHAR(255) NOT NULL,
    k JSON NOT NULL,
    n_0 INT NOT NULL,
    l INT NOT NULL,
    m INT NOT NULL,
    tpm_type VARCHAR(255) NOT NULL,
    learn_rule VARCHAR(255) NOT NULL,
    start_time DATETIME NOT NULL,
    end_time DATETIME NOT NULL,
    status VARCHAR(255) NOT NULL,
    stimulate_iterations INT NOT NULL,
    learn_iterations INT NOT NULL,
    final_weights JSON NOT NULL
);