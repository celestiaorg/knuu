# Overview

Thank you for taking the time to contribute to Knuu. All contributions are welcome and appreciated, even the tiniest fixes! :)

If you are new to Knuu, please go through the [README](./README.md) to familiarise yourself with the project.

## Choosing an issue to work on

- Notice something missing in the repo? Open a new [issue](https://github.com/celestiaorg/knuu/issues).

- Have a suggestion you want to implement? Open an issue, then create a Pull Request with your implementation.

- Want to resolve an issue in the repo, make a request to be assigned, and be sure to tag the issue in your PR!

## Contribution Workflow

To contribute to Knuu, follow these steps:

1. Fork the repository: Click on the 'Fork' button at the top right of this page. This will create a copy of the repository in your GitHub account.

1. Clone the forked repository to your local machine. Open your terminal and run:

	```bash
	git clone <forked-repository-url>
	```

1. Create your working branch: In the project directory, create a new branch by running the following command in your terminal:

	```bash
	git checkout -b working-branch
	```

	Be sure to name your branch according to the changes you are making.
	For example: `add-missing-tests`.

1. Make your changes: Do not address multiple issues per PR.
	For example, if you are adding a feature, it should not have bug fixes too. This is to enable the maintainers review your PR efficiently.

1. Commit the changes to your branch: After making the desired changes to the repo, run the following commands to commit them:

	```bash
	git add .
	```

	```bash
	git commit -m "add an appropriate commit message"
	```

	```bash
	git push origin <working-branch>
	```

	To ensure that your contribution is working as expected, please run `knuu-example` with your fork and working branch.

1. Create a Pull Request: Go to your forked repository on GitHub, and be sure that you are in the branch you pushed the changes to. Click on the 'Compare & pull request' button. This will open a new page where you can create your PR. Fill in the description field and click on 'Create pull request'.
Be sure to name your PR with a semantic prefix. For example, if it is a fix, it should be specified with the `fix:` prefix.

Congratulations! You have successfully made a Pull Request, and your changes will be reviewed by the maintainers.
