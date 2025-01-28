# EPURER - Mac Cleanup Tool

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

EPURER is a comprehensive and interactive tool designed to clean up your MacBook. It provides a modular approach with support for multiple languages (English and French) and asks for user confirmation before performing critical actions.

## Features

- **Interactive Cleanup**: Prompts user confirmation before each major cleanup action.
- **Multi-language Support**: Supports both English (`en_US`) and French (`fr_FR`).
- **Modular Design**: Organized into separate functions for better maintainability.
- **Comprehensive Cleanup**: Covers various aspects of system cleaning including trash, caches, logs, temporary files, DNS cache, Xcode data, Homebrew cache, localizations, iOS backups, and Launchpad database.

## Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/0sansnom/EPURER.git
   cd EPURER
   ```

2. Make the script executable:
   ```bash
   chmod +x epurer.sh
   ```

## Usage

To run the script in default language (French):
```bash
./epurer.sh
```

To specify a different language, set the `LANG` environment variable before running the script:
- For English:
  ```bash
  LANG=en_US ./epurer.sh
  ```
- For French:
  ```bash
  LANG=fr_FR ./epurer.sh
  ```

## Project Structure

```
EPURER/
├── epurer.sh                # Main script
├── languages/
│   ├── en_US.lang           # English language file
│   └── fr_FR.lang           # French language file
└── functions/
    ├── common_functions.sh  # Common utility functions
    ├── confirmation_functions.sh # Confirmation prompt functions
    └── cleanup_functions.sh # Cleanup specific functions
```

## Functions

### Common Functions
- **common_functions.sh**: Handles loading of language files and provides color-coded message printing.

### Confirmation Functions
- **confirmation_functions.sh**: Provides functionality to ask for user confirmation before executing critical actions.

### Cleanup Functions
- **cleanup_functions.sh**: Contains all the specific cleanup functions such as clearing caches, deleting logs, removing temporary files, etc.

## Contributing

Contributions are welcome! If you have any suggestions, improvements, or bug fixes, please feel free to open an issue or submit a pull request.

1. Fork the repository.
2. Create a new branch (`git checkout -b feature-branch`).
3. Commit your changes (`git commit -am 'Add some feature'`).
4. Push to the branch (`git push origin feature-branch`).
5. Open a pull request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
