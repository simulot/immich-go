# Contributing to immich-go

Hello, and thank you for your interest in contributing to `immich-go`! Your help is vital for the health and growth of this project. To ensure a smooth and effective collaboration, please follow the guidelines outlined in this document.

By following these rules, you help us maintain a clean, stable, and well-organized codebase.

## Getting Started

To get started with your contribution, please follow the standard GitHub workflow:

1.  **Fork the Repository:** Start by forking the `simulot/immich-go` repository to your own GitHub account. This creates a personal copy of the project where you can work freely.
2.  **Clone your Fork:** Clone your fork to your local machine to begin development. You can do this using the Git command line:
    ```sh
    git clone https://github.com/<your-username>/immich-go.git
    cd immich-go
    ```
3.  **Add the Original Repository as a Remote:** To stay up to date with the main project, add the original repository as an "upstream" remote.
    ```sh
    git remote add upstream https://github.com/simulot/immich-go.git
    ```
    You can now use `git pull upstream next` to fetch the latest changes from the main development branch.

## Our Git Branching Model

Our repository uses a structured branching model to manage development and releases effectively.

  * **`main`:** This branch always contains the code for the latest official release. It should be considered stable and ready for production at all times. All new code is merged into `main` only from `hotfix` or `next` branches.
  * **`next`:** This is our primary development branch. All new features and regular bug fixes are integrated here. It represents the state of the project for the upcoming release.
  * **`hotfix/*`:** Short-lived branches used for urgent bug fixes that must be applied directly to the latest release on `main`. These are always created from `main`.
  * **`feature/*`** and **`bugfix/*`:** Short-lived branches for developing new features or fixing non-urgent bugs. These are always created from `next`.

## Your Contribution Workflow

Your workflow depends on the nature of your contribution:

### 1. For a New Feature or a Regular Bug Fix

For all non-urgent changes, your work should be based on the `next` branch.

1.  **Sync with `next`:** Ensure your local `next` branch is up to date with the latest changes from the main repository.
    ```sh
    git checkout next
    git pull upstream next
    ```
2.  **Create a New Branch:** Create a new branch for your work using a descriptive name that follows our convention:
      * For features: `feature/your-feature-name`
      * For bug fixes: `bugfix/your-bug-description`
    ```sh
    git checkout -b feature/my-new-feature
    ```
3.  **Develop and Commit:** Make your changes, test them, and commit your work. Use clear and descriptive commit messages.
4.  **Push your Branch:** Push your new branch to your personal fork on GitHub.
    ```sh
    git push origin feature/my-new-feature
    ```
5.  **Create a Pull Request:** Go to your fork on GitHub and open a new Pull Request. The **base branch must be `next`**. Your Pull Request will automatically be checked by our Continuous Integration (CI) system to ensure it meets our quality standards.

### 2\. For an Urgent Hotfix

A hotfix is a critical bug that needs to be fixed in the current production version. This process is handled with extra care.

1.  **Sync with `main`:** Ensure your local `main` branch is up to date.
    ```sh
    git checkout main
    git pull upstream main
    ```
2.  **Create a New Branch:** Create a hotfix branch from `main` using a descriptive name:
      * For a hotfix: `hotfix/critical-bug`
    ```sh
    git checkout -b hotfix/critical-bug
    ```
3.  **Develop, Commit, and Push:** Make your changes, commit them, and push your hotfix branch to your fork.
4.  **Create a Pull Request:** Open a new Pull Request on GitHub. The **base branch must be `main`**. Our CI/CD pipeline will automatically run to validate the fix.

## Pull Request Guidelines

To make the review process as efficient as possible, please follow these guidelines when creating a Pull Request:

  * **Descriptive Title and Body:** Provide a clear and concise title for your PR. In the description, explain the purpose of your changes, the problem they solve, and any relevant context.
  * **Pass CI/CD Checks:** All Pull Requests must have a passing status from our automated checks before they can be merged. These checks include building the project and running tests.
  * **Target the Right Branch:** Double-check that you are opening the PR to the correct target branch (`next` for new features/bugfixes, `main` for hotfixes). Our automated system will block incorrect merges.
  * **Code Style:** Please follow the existing code style.

Thank you for your contribution!

